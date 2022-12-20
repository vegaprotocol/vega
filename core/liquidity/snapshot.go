// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package liquidity

import (
	"context"
	"sort"
	"strconv"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/orderbook_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity OrderBook
type OrderBook interface {
	GetOrderByID(orderID string) (*types.Order, error)
	GetOrdersPerParty(party string) []*types.Order
	GetLiquidityOrders(party string) []*types.Order
}

type SnapshotEngine struct {
	*Engine
	pl     types.Payload
	market string

	// liquidity types
	stopped                     bool
	serialisedParameters        []byte
	serialisedPendingProvisions []byte
	serialisedProvisions        []byte
	serialisedSupplied          []byte
	serialisedScores            []byte

	// keys, need to be computed when the engine is
	// instantiated as they are dynamic
	hashKeys             []string
	parametersKey        string
	pendingProvisionsKey string
	provisionsKey        string
	suppliedKey          string
	scoresKey            string
}

func NewSnapshotEngine(config Config,
	log *logging.Logger,
	timeService TimeService,
	broker Broker,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	orderBook OrderBook,
	asset string,
	market string,
	stateVarEngine StateVarEngine,
	priceFactor *num.Uint,
	positionFactor num.Decimal,
) *SnapshotEngine {
	se := &SnapshotEngine{
		// tickSize = 10^{market_dp} - used for calculating probabilities at offsets from the best bid/ask
		// priceFactor = 10^{asset_dp} / 10^{market_dp} - used for scaling a price to the market
		// positionFactor = 10^{position_dp} - used to scale sizes to the market position decimals
		Engine:  NewEngine(config, log, timeService, broker, riskModel, priceMonitor, orderBook, asset, market, stateVarEngine, priceFactor, positionFactor),
		pl:      types.Payload{},
		market:  market,
		stopped: false,
	}

	// build the keys
	se.buildHashKeys(market)

	return se
}

func (e *SnapshotEngine) UpdateMarketConfig(model risk.Model, monitor PriceMonitor) {
	e.Engine.UpdateMarketConfig(model, monitor)
}

func (e *SnapshotEngine) StopSnapshots() {
	e.log.Debug("market has been cleared, stopping snapshot production", logging.String("marketid", e.marketID))
	e.stopped = true
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.LiquiditySnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return e.hashKeys
}

func (e *SnapshotEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshotEngine) OnSuppliedStakeToObligationFactorUpdate(v num.Decimal) {
	e.Engine.OnSuppliedStakeToObligationFactorUpdate(v)
}

func (e *SnapshotEngine) OnMaximumLiquidityFeeFactorLevelUpdate(f num.Decimal) {
	e.Engine.OnMaximumLiquidityFeeFactorLevelUpdate(f)
}

func (e *SnapshotEngine) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	return e.Engine.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
}

func (e *SnapshotEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *SnapshotEngine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadLiquidityPendingProvisions:
		return nil, e.loadPendingProvisions(
			ctx, pl.PendingProvisions.GetPendingProvisions(), p)
	case *types.PayloadLiquidityProvisions:
		return nil, e.loadProvisions(
			ctx, pl.Provisions.GetLiquidityProvisions(), p)
	case *types.PayloadLiquidityParameters:
		return nil, e.loadParameters(ctx, pl.Parameters, p)
	case *types.PayloadLiquiditySupplied:
		return nil, e.loadSupplied(pl.LiquiditySupplied, p)
	case *types.PayloadLiquidityScores:
		return nil, e.loadLiquidityScores(pl.LiquidityScores, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) loadSupplied(ls *snapshotpb.LiquiditySupplied, p *types.Payload) error {
	err := e.suppliedEngine.Reload(ls)
	if err != nil {
		return err
	}
	e.serialisedSupplied, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *SnapshotEngine) loadLiquidityScores(ls *snapshotpb.LiquidityScores, p *types.Payload) error {
	var err error
	e.nAvg = int64(ls.RunningAverageCounter)

	scores := make(map[string]num.Decimal, len(ls.Scores))
	for _, p := range ls.Scores {
		score, err := num.DecimalFromString(p.Score)
		if err != nil {
			return err
		}
		scores[p.PartyId] = score
	}

	e.avgScores = scores

	e.serialisedScores, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *SnapshotEngine) loadParameters(
	_ context.Context, parameters *snapshotpb.LiquidityParameters, p *types.Payload,
) error {
	maxShapesSize, err := strconv.ParseInt(parameters.MaxShapeSize, 10, 64)
	if err != nil {
		return err
	}
	if err := e.OnMarketLiquidityProvisionShapesMaxSizeUpdate(maxShapesSize); err != nil {
		return err
	}

	maxFee, err := num.DecimalFromString(parameters.MaxFee)
	if err != nil {
		return err
	}
	e.OnMaximumLiquidityFeeFactorLevelUpdate(maxFee)

	stof, err := num.DecimalFromString(parameters.StakeToObligationFactor)
	if err != nil {
		return err
	}
	e.OnSuppliedStakeToObligationFactorUpdate(stof)
	e.serialisedParameters, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *SnapshotEngine) loadPendingProvisions(
	_ context.Context, pendingProvisions []string, p *types.Payload,
) error {
	e.Engine.pendings = newSnapshotablePendingProvisions()
	for _, v := range pendingProvisions {
		e.Engine.pendings.Add(v)
	}
	var err error
	e.serialisedPendingProvisions, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *SnapshotEngine) loadProvisions(
	ctx context.Context, provisions []*typespb.LiquidityProvision, p *types.Payload,
) error {
	e.Engine.provisions = newSnapshotableProvisionsPerParty()
	evts := make([]events.Event, 0, len(provisions))
	for _, v := range provisions {
		provision, err := types.LiquidityProvisionFromProto(v)
		if err != nil {
			return err
		}
		e.Engine.provisions.Set(v.PartyId, provision)
		evts = append(evts, events.NewLiquidityProvisionEvent(ctx, provision))
	}

	var err error
	e.serialisedProvisions, err = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	return err
}

func (e *SnapshotEngine) serialise(k string) ([]byte, error) {
	var (
		buf     []byte
		changed bool
		err     error
	)
	switch k {
	case e.parametersKey:
		buf, changed, err = e.serialiseParameters()
	case e.pendingProvisionsKey:
		buf, changed, err = e.serialisePendingProvisions()
	case e.provisionsKey:
		buf, changed, err = e.serialiseProvisions()
	case e.suppliedKey:
		buf, changed, err = e.serialiseSupplied()
	case e.scoresKey:
		buf, changed, err = e.serialiseScores()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if err != nil {
		return nil, err
	}

	if e.stopped {
		return nil, nil
	}

	if !changed {
		return buf, nil
	}

	switch k {
	case e.parametersKey:
		e.serialisedParameters = buf
	case e.pendingProvisionsKey:
		e.serialisedPendingProvisions = buf
	case e.provisionsKey:
		e.serialisedProvisions = buf
	case e.suppliedKey:
		e.serialisedSupplied = buf
	case e.scoresKey:
		e.serialisedScores = buf
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	return buf, nil
}

func (e *SnapshotEngine) serialiseParameters() ([]byte, bool, error) {
	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityParameters{
			LiquidityParameters: &snapshotpb.LiquidityParameters{
				MaxFee:                  e.Engine.maxFee.String(),
				MaxShapeSize:            strconv.Itoa(int(e.Engine.maxShapesSize)),
				StakeToObligationFactor: e.Engine.stakeToObligationFactor.String(),
				MarketId:                e.market,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *SnapshotEngine) serialisePendingProvisions() ([]byte, bool, error) {
	pbpendings := make([]string, 0, len(e.Engine.pendings.m))
	for k := range e.Engine.pendings.m {
		pbpendings = append(pbpendings, k)
	}
	sort.Strings(pbpendings)

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityPendingProvisions{
			LiquidityPendingProvisions: &snapshotpb.LiquidityPendingProvisions{
				MarketId:          e.market,
				PendingProvisions: pbpendings,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *SnapshotEngine) serialiseProvisions() ([]byte, bool, error) {
	if len(e.lpPartyOrders) != 0 {
		e.log.Panic("lp orders exist in engine during snapshot, they should only exist in a transitionary period during a redeploy")
	}

	// these are sorted already, only a conversion to proto is needed
	lps := e.Engine.provisions.Slice()
	pblps := make([]*typespb.LiquidityProvision, 0, len(lps))
	for _, v := range lps {
		pblps = append(pblps, v.IntoProto())
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityProvisions{
			LiquidityProvisions: &snapshotpb.LiquidityProvisions{
				MarketId:            e.market,
				LiquidityProvisions: pblps,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *SnapshotEngine) serialiseSupplied() ([]byte, bool, error) {
	payload := e.suppliedEngine.Payload()
	return e.marshalPayload(payload)
}

func (e *SnapshotEngine) serialiseScores() ([]byte, bool, error) {
	scores := make([]*snapshotpb.LiquidityScore, 0, len(e.avgScores))

	keys := make([]string, 0, len(e.avgScores))
	for k := range e.avgScores {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		s := &snapshotpb.LiquidityScore{
			PartyId: k,
			Score:   e.avgScores[k].String(),
		}
		scores = append(scores, s)
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityScores{
			LiquidityScores: &snapshotpb.LiquidityScores{
				MarketId:              e.market,
				RunningAverageCounter: int32(e.nAvg),
				Scores:                scores,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *SnapshotEngine) marshalPayload(payload *snapshotpb.Payload) ([]byte, bool, error) {
	buf, err := proto.Marshal(payload)
	if err != nil {
		return nil, false, err
	}

	return buf, true, nil
}

func (e *SnapshotEngine) buildHashKeys(market string) {
	e.parametersKey = (&types.PayloadLiquidityParameters{
		Parameters: &snapshotpb.LiquidityParameters{
			MarketId: market,
		},
	}).Key()
	e.pendingProvisionsKey = (&types.PayloadLiquidityPendingProvisions{
		PendingProvisions: &snapshotpb.LiquidityPendingProvisions{
			MarketId: market,
		},
	}).Key()
	e.provisionsKey = (&types.PayloadLiquidityProvisions{
		Provisions: &snapshotpb.LiquidityProvisions{
			MarketId: market,
		},
	}).Key()

	e.suppliedKey = (&types.PayloadLiquiditySupplied{
		LiquiditySupplied: &snapshotpb.LiquiditySupplied{
			MarketId: market,
		},
	}).Key()

	e.scoresKey = (&types.PayloadLiquidityScores{
		LiquidityScores: &snapshotpb.LiquidityScores{
			MarketId: market,
		},
	}).Key()

	e.hashKeys = append([]string{}, e.parametersKey,
		e.pendingProvisionsKey, e.provisionsKey, e.suppliedKey, e.scoresKey)
}
