package liquidity

import (
	"context"
	"sort"
	"strconv"

	typespb "code.vegaprotocol.io/protos/vega"
	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type SnapshotEngine struct {
	*Engine
	pl     types.Payload
	market string

	// liquidity types
	parametersChanged                bool
	stopped                          bool
	serialisedParameters             []byte
	serialisedPartiesLiquidityOrders []byte
	serialisedPartiesOrders          []byte
	serialisedPendingProvisions      []byte
	serialisedProvisions             []byte
	serialisedSupplied               []byte

	// keys, need to be computed when the engine is
	// instantiated as they are dynamic
	hashKeys                  []string
	parametersKey             string
	partiesLiquidityOrdersKey string
	partiesOrdersKey          string
	pendingProvisionsKey      string
	provisionsKey             string
	suppliedKey               string
}

func NewSnapshotEngine(config Config,
	log *logging.Logger,
	broker Broker,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	asset string,
	market string,
	stateVarEngine StateVarEngine,
	tickSize *num.Uint,
	priceFactor *num.Uint,
	positionFactor num.Decimal,
) *SnapshotEngine {
	se := &SnapshotEngine{
		// tickSize = 10^{market_dp} - used for calculating probabilities at offsets from the best bid/ask
		// priceFactor = 10^{asset_dp} / 10^{market_dp} - used for scaling a price to the market
		// positionFactor = 10^{position_dp} - used to scale sizes to the market position decimals
		Engine: NewEngine(config, log, broker, riskModel, priceMonitor, asset, market, stateVarEngine, tickSize, priceFactor, positionFactor),
		pl:     types.Payload{},
		market: market,

		parametersChanged: true,
		stopped:           false,
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

func (e *SnapshotEngine) Changed() bool {
	return e.parametersChanged
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
	e.parametersChanged = true
	e.Engine.OnSuppliedStakeToObligationFactorUpdate(v)
}

func (e *SnapshotEngine) OnMaximumLiquidityFeeFactorLevelUpdate(f num.Decimal) {
	e.parametersChanged = true
	e.Engine.OnMaximumLiquidityFeeFactorLevelUpdate(f)
}

func (e *SnapshotEngine) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	e.parametersChanged = true
	return e.Engine.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
}

func (e *SnapshotEngine) HasChanged(k string) bool {
	switch k {
	case e.parametersKey:
		return e.parametersChanged
	case e.partiesLiquidityOrdersKey:
		return e.Engine.liquidityOrders.HasUpdates()
	case e.partiesOrdersKey:
		return e.Engine.orders.HasUpdates()
	case e.pendingProvisionsKey:
		return e.Engine.pendings.HasUpdates()
	case e.provisionsKey:
		return e.Engine.provisions.HasUpdates()
	case e.suppliedKey:
		return e.suppliedEngine.HasUpdates()
	default:
		return false
	}
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
	case *types.PayloadLiquidityPartiesOrders:
		return nil, e.loadPartiesOrders(ctx, pl.PartiesOrders.GetOrders(), p)
	case *types.PayloadLiquidityPartiesLiquidityOrders:
		return nil, e.loadPartiesLiquidityOrders(
			ctx, pl.PartiesLiquidityOrders.GetOrders(), p)
	case *types.PayloadLiquiditySupplied:
		return nil, e.loadSupplied(pl.LiquiditySupplied, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) loadSupplied(ls *snapshotpb.LiquiditySupplied, p *types.Payload) error {
	err := e.suppliedEngine.Reload(ls)
	if err != nil {
		return err
	}
	e.Engine.suppliedEngine.ResetUpdated()
	e.serialisedSupplied, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *SnapshotEngine) loadPartiesOrders(
	_ context.Context, orders []*typespb.Order, p *types.Payload,
) error {
	e.Engine.orders = newSnapshotablePartiesOrders()
	for _, v := range orders {
		order, err := types.OrderFromProto(v)
		if err != nil {
			return err
		}
		e.Engine.orders.Add(order.Party, order)
	}
	var err error
	e.Engine.orders.ResetUpdated()
	e.serialisedPartiesOrders, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *SnapshotEngine) loadPartiesLiquidityOrders(
	_ context.Context, orders []*typespb.Order, p *types.Payload,
) error {
	e.Engine.liquidityOrders = newSnapshotablePartiesOrders()
	for _, v := range orders {
		order, err := types.OrderFromProto(v)
		if err != nil {
			return err
		}
		e.Engine.liquidityOrders.Add(order.Party, order)
	}
	var err error
	e.Engine.liquidityOrders.ResetUpdated()
	e.serialisedPartiesLiquidityOrders, err = proto.Marshal(p.IntoProto())
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
	e.parametersChanged = false
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
	e.Engine.pendings.ResetUpdated()
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
	e.Engine.provisions.ResetUpdated()
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
	case e.partiesLiquidityOrdersKey:
		buf, changed, err = e.serialisePartiesLiquidityOrders()
	case e.partiesOrdersKey:
		buf, changed, err = e.serialisePartiesOrders()
	case e.pendingProvisionsKey:
		buf, changed, err = e.serialisePendingProvisions()
	case e.provisionsKey:
		buf, changed, err = e.serialiseProvisions()
	case e.suppliedKey:
		buf, changed, err = e.serialiseSupplied()
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
	case e.partiesLiquidityOrdersKey:
		e.serialisedPartiesLiquidityOrders = buf
	case e.partiesOrdersKey:
		e.serialisedPartiesOrders = buf
	case e.pendingProvisionsKey:
		e.serialisedPendingProvisions = buf
	case e.provisionsKey:
		e.serialisedProvisions = buf
	case e.suppliedKey:
		e.serialisedSupplied = buf
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	return buf, nil
}

func (e *SnapshotEngine) serialiseParameters() ([]byte, bool, error) {
	key := e.parametersKey
	if !e.parametersChanged {
		return e.serialisedParameters, false, nil
	}

	// reset the flag
	e.parametersChanged = false

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

	return e.marshalPayload(key, payload)
}

func (e *SnapshotEngine) serialisePartiesLiquidityOrders() ([]byte, bool, error) {
	key := e.partiesLiquidityOrdersKey
	if !e.Engine.liquidityOrders.HasUpdates() {
		return e.serialisedPartiesLiquidityOrders, false, nil
	}

	e.Engine.liquidityOrders.ResetUpdated()

	pborders := []*typespb.Order{}
	for _, orders := range e.Engine.liquidityOrders.m {
		for _, order := range orders {
			pborders = append(pborders, order.IntoProto())
		}
	}
	sort.SliceStable(pborders, func(i, j int) bool { return pborders[i].Id < pborders[j].Id })

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityPartiesLiquidityOrders{
			LiquidityPartiesLiquidityOrders: &snapshotpb.LiquidityPartiesLiquidityOrders{
				MarketId: e.market,
				Orders:   pborders,
			},
		},
	}

	return e.marshalPayload(key, payload)
}

func (e *SnapshotEngine) serialisePartiesOrders() ([]byte, bool, error) {
	key := e.partiesOrdersKey
	if !e.Engine.orders.HasUpdates() {
		return e.serialisedPartiesOrders, false, nil
	}

	e.Engine.orders.ResetUpdated()

	pborders := []*typespb.Order{}
	for _, orders := range e.Engine.orders.m {
		for _, order := range orders {
			pborders = append(pborders, order.IntoProto())
		}
	}
	sort.SliceStable(pborders, func(i, j int) bool { return pborders[i].Id < pborders[j].Id })

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityPartiesOrders{
			LiquidityPartiesOrders: &snapshotpb.LiquidityPartiesOrders{
				MarketId: e.market,
				Orders:   pborders,
			},
		},
	}

	return e.marshalPayload(key, payload)
}

func (e *SnapshotEngine) serialisePendingProvisions() ([]byte, bool, error) {
	key := e.pendingProvisionsKey
	if !e.Engine.pendings.HasUpdates() {
		return e.serialisedPendingProvisions, false, nil
	}

	e.Engine.pendings.ResetUpdated()

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

	return e.marshalPayload(key, payload)
}

func (e *SnapshotEngine) serialiseProvisions() ([]byte, bool, error) {
	key := e.provisionsKey
	if !e.Engine.provisions.HasUpdates() {
		return e.serialisedProvisions, false, nil
	}

	e.Engine.provisions.ResetUpdated()

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

	return e.marshalPayload(key, payload)
}

func (e *SnapshotEngine) serialiseSupplied() ([]byte, bool, error) {
	key := e.suppliedKey
	if !e.suppliedEngine.HasUpdates() {
		return e.serialisedSupplied, false, nil
	}

	e.suppliedEngine.ResetUpdated()

	payload := e.suppliedEngine.Payload()
	return e.marshalPayload(key, payload)
}

func (e *SnapshotEngine) marshalPayload(key string, payload *snapshotpb.Payload) ([]byte, bool, error) {
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
	e.partiesLiquidityOrdersKey = (&types.PayloadLiquidityPartiesLiquidityOrders{
		PartiesLiquidityOrders: &snapshotpb.LiquidityPartiesLiquidityOrders{
			MarketId: market,
		},
	}).Key()
	e.partiesOrdersKey = (&types.PayloadLiquidityPartiesOrders{
		PartiesOrders: &snapshotpb.LiquidityPartiesOrders{
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

	e.hashKeys = append([]string{}, e.parametersKey, e.partiesLiquidityOrdersKey,
		e.partiesOrdersKey, e.pendingProvisionsKey, e.provisionsKey, e.suppliedKey)
}
