// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package amm

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var (
	ErrNoPoolMatchingParty  = errors.New("no pool matching party")
	ErrPartyAlreadyOwnAPool = func(market string) error {
		return fmt.Errorf("party already own a pool for market %v", market)
	}
	ErrCommitmentTooLow          = errors.New("commitment amount too low")
	ErrRebaseOrderDidNotTrade    = errors.New("rebase-order did not trade")
	ErrRebaseTargetOutsideBounds = errors.New("rebase target outside bounds")
)

const (
	version = "AMMv1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/amm Collateral,Position,Market,Risk

type Collateral interface {
	GetAssetQuantum(asset string) (num.Decimal, error)
	GetPartyMarginAccount(market, party, asset string) (*types.Account, error)
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
	SubAccountUpdate(
		ctx context.Context,
		party, subAccount, asset, market string,
		transferType types.TransferType,
		amount *num.Uint,
	) (*types.LedgerMovement, error)
	SubAccountRelease(
		ctx context.Context,
		party, subAccount, asset, market string, mevt events.MarketPosition,
	) ([]*types.LedgerMovement, events.Margin, error)
	CreatePartyAMMsSubAccounts(
		ctx context.Context,
		party, subAccount, asset, market string,
	) (general *types.Account, margin *types.Account, err error)
}

type Broker interface {
	Send(events.Event)
}

type Market interface {
	GetID() string
	ClosePosition(context.Context, string) bool // return true if position was successfully closed
	GetSettlementAsset() string
	SubmitOrderWithIDGeneratorAndOrderID(context.Context, *types.OrderSubmission, string, common.IDGenerator, string, bool) (*types.OrderConfirmation, error)
}

type Risk interface {
	GetRiskFactors() *types.RiskFactor
	GetScalingFactors() *types.ScalingFactors
	GetSlippage() num.Decimal
}

type Position interface {
	GetPositionsByParty(ids ...string) []events.MarketPosition
}

type sqrtFn func(*num.Uint) num.Decimal

// Sqrter calculates sqrt's of Uints and caches the results. We want this cache to be shared across all pools for a market.
type Sqrter struct {
	cache map[string]num.Decimal
}

// sqrt calculates the square root of the uint and caches it.
func (s *Sqrter) sqrt(u *num.Uint) num.Decimal {
	if r, ok := s.cache[u.String()]; ok {
		return r
	}

	// TODO that we may need to re-visit this depending on the performance impact
	// but for now lets do it "properly" in full decimals and work out how we can
	// improve it once we have reg-tests and performance data.

	// integer sqrt is a good approximation
	r := num.UintOne().Sqrt(u).ToDecimal()

	// so now lets do a few iterations using Heron's Method to get closer
	// x_i = (x + u/x) / 2
	ud := u.ToDecimal()
	for i := 0; i < 6; i++ {
		r = r.Add(ud.Div(r)).Div(num.DecimalFromInt64(2))
	}

	// and cache it -- we can also maybe be more clever here and use a LRU but thats for later
	s.cache[u.String()] = r
	return r
}

type Engine struct {
	log *logging.Logger

	broker Broker

	risk       Risk
	collateral Collateral
	position   Position
	market     Market
	idgen      *idgeneration.IDGenerator

	// gets us from the price in the submission -> price in full asset dp
	priceFactor    *num.Uint
	positionFactor num.Decimal

	// map of party -> pool
	pools    map[string]*Pool
	poolsCpy []*Pool

	// sqrt calculator with cache
	rooter *Sqrter

	// a mapping of all sub accounts to the party owning them.
	subAccounts map[string]string

	minCommitmentQuantum *num.Uint
}

func New(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	market Market,
	risk Risk,
	position Position,
	priceFactor *num.Uint,
	positionFactor num.Decimal,
) *Engine {
	return &Engine{
		log:                  log,
		broker:               broker,
		risk:                 risk,
		collateral:           collateral,
		position:             position,
		market:               market,
		pools:                map[string]*Pool{},
		poolsCpy:             []*Pool{},
		subAccounts:          map[string]string{},
		minCommitmentQuantum: num.UintZero(),
		rooter:               &Sqrter{cache: map[string]num.Decimal{}},
		priceFactor:          priceFactor,
		positionFactor:       positionFactor,
	}
}

func NewFromProto(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	market Market,
	risk Risk,
	position Position,
	state *v1.AmmState,
	priceFactor *num.Uint,
	positionFactor num.Decimal,
) *Engine {
	e := New(log, broker, collateral, market, risk, position, priceFactor, positionFactor)

	for _, v := range state.SubAccounts {
		e.subAccounts[v.Key] = v.Value
	}

	// TODO consider whether we want the cache in the snapshot, it might be pretty large/slow and I'm not sure what we gain
	for _, v := range state.Sqrter {
		e.rooter.cache[v.Key] = num.MustDecimalFromString(v.Value)
	}

	for _, v := range state.Pools {
		e.add(NewPoolFromProto(e.rooter.sqrt, e.collateral, e.position, v.Pool, v.Party, priceFactor))
	}

	return e
}

func (e *Engine) IntoProto() *v1.AmmState {
	state := &v1.AmmState{
		Sqrter:      make([]*v1.StringMapEntry, 0, len(e.rooter.cache)),
		SubAccounts: make([]*v1.StringMapEntry, 0, len(e.subAccounts)),
		Pools:       make([]*v1.PoolMapEntry, 0, len(e.pools)),
	}

	for k, v := range e.rooter.cache {
		state.Sqrter = append(state.Sqrter, &v1.StringMapEntry{
			Key:   k,
			Value: v.String(),
		})
	}
	sort.Slice(state.Sqrter, func(i, j int) bool { return state.Sqrter[i].Key < state.Sqrter[j].Key })

	for k, v := range e.subAccounts {
		state.SubAccounts = append(state.SubAccounts, &v1.StringMapEntry{
			Key:   k,
			Value: v,
		})
	}
	sort.Slice(state.SubAccounts, func(i, j int) bool { return state.SubAccounts[i].Key < state.SubAccounts[j].Key })

	for _, v := range e.poolsCpy {
		state.Pools = append(state.Pools, &v1.PoolMapEntry{
			Party: v.party,
			Pool:  v.IntoProto(),
		})
	}
	return state
}

func (e *Engine) OnMinCommitmentQuantumUpdate(ctx context.Context, c *num.Uint) {
	e.minCommitmentQuantum = c.Clone()
}

func (e *Engine) OnTick(ctx context.Context, _ time.Time) {
	// check sub account balances (margin, general)

	// seed an id-generator to create IDs for any orders generated in this block
	_, blockHash := vgcontext.TraceIDFromContext(ctx)
	e.idgen = idgeneration.New(blockHash + crypto.HashStrToHex("amm-engine"+e.market.GetID()))

	// any pools that are closing that are at position 0 should be removed completely
	for _, p := range e.poolsCpy {
		if p.closing() {
			if pos, _ := p.getPosition(); pos == 0 {
				if _, err := e.releaseSubAccounts(ctx, p); err != nil {
					e.log.Error("unable to release subaccount balance", logging.Error(err))
				}
				p.status = types.AMMPoolStatusCancelled
				e.remove(ctx, p.party)
			}
		}
	}

	// TODO check sub account balances (margin, general)
}

func (e *Engine) IsPoolSubAccount(key string) bool {
	_, yes := e.subAccounts[key]
	return yes
}

// SubmitOrder takes an aggressive order and generates matching orders with the registered AMMs such that
// volume is only taken in the interval (inner, outer) where inner and outer are price-levels on the orderbook.
func (e *Engine) SubmitOrder(agg *types.Order, inner, outer *num.Uint) []*types.Order {
	if len(e.pools) == 0 {
		return nil
	}

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("looking for match with order",
			logging.Int("n-pools", len(e.pools)),
			logging.Order(agg),
		)
		e.log.Debug("between prices",
			logging.String("inner", inner.String()),
			logging.String("outer", outer.String()),
		)
	}

	active := []*Pool{}
	orders := []*types.Order{}
	best := outer.Clone()

	// first we find all amm's whose best-price would allow a trade with the incoming order
	for _, p := range e.poolsCpy {
		// if pool is in reducing only mode and order will increase its position, we don't want to trade
		if !p.canTrade(agg) {
			continue
		}
		p.setEphemeralPosition()

		price := p.BestPrice(agg)
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("best price for pool",
				logging.String("id", p.ID),
				logging.String("best-price", price.String()),
			)
		}

		if agg.Side == types.SideBuy {
			if price.GTE(outer) || price.GT(agg.Price) {
				// either fair price is out of bounds, or is selling at higher than incoming buy
				continue
			}
			active = append(active, p)
			best = num.Min(best, p.upper.high)
		}

		if agg.Side == types.SideSell {
			if price.LTE(outer) || price.LT(agg.Price) {
				// either fair price is out of bounds, or is buying at lower than incoming sell
				continue
			}
			active = append(active, p)
			best = num.Max(best, p.lower.low)
		}
	}

	if agg.Side == types.SideSell {
		inner, best = best, inner
	}

	// calculate the volume each pool has
	var total uint64
	volumes := []uint64{}
	for _, p := range active {
		volume := p.VolumeBetweenPrices(agg.Side, inner, best)
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("volume available to trade",
				logging.String("id", p.ID),
				logging.Uint64("volume", volume),
			)
		}

		volumes = append(volumes, volume)
		total += volume
	}

	// if the pools consume the whole incoming order's volume, share it out pro-rata
	if agg.Remaining < total {
		var retotal uint64
		for i := range volumes {
			volumes[i] = agg.Remaining * volumes[i] / total
			retotal += volumes[i]
		}

		// any lost crumbs due to integer division is given to the first pool
		if d := agg.Remaining - retotal; d != 0 {
			volumes[0] += d
		}
	}

	// now generate offbook orders
	for i, p := range active {
		volume := volumes[i]
		if volume == 0 {
			continue
		}

		pos, ae := p.getPosition()
		x, y := p.virtualBalances(pos, ae, agg.Side)
		dx := num.DecimalFromInt64(int64(volume))

		// dy = x*y / (x - dx) - y
		// where y and x are the balances on either side of the pool, and dx is the change in volume
		// then the trade price is dy/dx
		dy := x.Mul(y).Div(x.Sub(dx)).Sub(y)
		price, _ := num.UintFromDecimal(dy.Div(dx))
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("generated order at price",
				logging.String("id", p.ID),
				logging.String("price", price.String()),
				logging.Int64("pos", pos),
				logging.String("average-entry", ae.String()),
				logging.String("y", y.String()),
				logging.String("x", x.String()),
				logging.String("dy", dy.String()),
				logging.String("dx", dx.String()),
			)
		}

		// construct the orders
		o := &types.Order{
			ID:               e.idgen.NextID(),
			MarketID:         p.market,
			Party:            p.SubAccount,
			Size:             volume,
			Remaining:        volume,
			Price:            price,
			OriginalPrice:    num.UintZero().Div(price, e.priceFactor),
			Side:             types.OtherSide(agg.Side),
			TimeInForce:      types.OrderTimeInForceFOK,
			Type:             types.OrderTypeMarket,
			CreatedAt:        agg.CreatedAt,
			Status:           types.OrderStatusFilled,
			Reference:        "vamm-" + p.SubAccount,
			GeneratedOffbook: true,
		}
		orders = append(orders, o)
		p.updateEphemeralPosition(o)
	}

	return orders
}

// NotifyFinished is called when the matching engine has finished matching an order and is returning it to
// the market for processing.
func (e *Engine) NotifyFinished() {
	for _, p := range e.poolsCpy {
		p.clearEphemeralPosition()
	}
}

func (e *Engine) SubmitAMM(
	ctx context.Context,
	submit *types.SubmitAMM,
	deterministicID string,
	targetPrice *num.Uint,
) error {
	idgen := idgeneration.New(deterministicID)
	poolID := idgen.NextID()

	subAccount := DeriveSubAccount(submit.Party, submit.MarketID, version, 0)
	_, ok := e.pools[submit.Party]
	if ok {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonPartyAlreadyOwnsAPool,
			),
		)

		return ErrPartyAlreadyOwnAPool(e.market.GetID())
	}

	if err := e.ensureCommitmentAmount(ctx, submit.CommitmentAmount); err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCommitmentTooLow,
			),
		)
		return err
	}

	_, _, err := e.collateral.CreatePartyAMMsSubAccounts(ctx, submit.Party, subAccount, e.market.GetSettlementAsset(), submit.MarketID)
	if err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonUnspecified,
			),
		)

		return err
	}

	err = e.updateSubAccountBalance(
		ctx, submit.Party, subAccount, submit.CommitmentAmount,
	)
	if err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCannotFillCommitment,
			),
		)

		return err
	}
	pool := NewPool(
		poolID,
		subAccount,
		e.market.GetSettlementAsset(),
		submit,
		e.rooter.sqrt,
		e.collateral,
		e.position,
		e.risk.GetRiskFactors(),
		e.risk.GetScalingFactors(),
		e.risk.GetSlippage(),
		e.priceFactor,
		e.positionFactor,
	)

	if targetPrice != nil {
		if err := e.rebasePool(ctx, pool, targetPrice, submit.SlippageTolerance, idgen); err != nil {
			if err := e.updateSubAccountBalance(ctx, submit.Party, subAccount, num.UintZero()); err != nil {
				e.log.Panic("unable to remove sub account balances", logging.Error(err))
			}

			// couldn't rebase the pool so it gets rejected
			e.broker.Send(
				events.NewAMMPoolEvent(
					ctx, submit.Party, e.market.GetID(), subAccount, poolID,
					submit.CommitmentAmount, submit.Parameters,
					types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCannotRebase,
				),
			)
			return err
		}
	}
	e.add(pool)
	e.sendUpdate(ctx, pool)
	return nil
}

func (e *Engine) AmendAMM(
	ctx context.Context,
	amend *types.AmendAMM,
	deterministicID string,
) error {
	pool, ok := e.pools[amend.Party]
	if !ok {
		return ErrNoPoolMatchingParty
	}

	if err := e.ensureCommitmentAmount(ctx, amend.CommitmentAmount); err != nil {
		return err
	}

	fairPrice := pool.BestPrice(nil)
	oldCommitment := pool.Commitment.Clone()

	err := e.updateSubAccountBalance(
		ctx, amend.Party, pool.SubAccount, amend.CommitmentAmount,
	)
	if err != nil {
		return err
	}

	pool.Update(amend, e.risk.GetRiskFactors(), e.risk.GetScalingFactors(), e.risk.GetSlippage())
	if err := e.rebasePool(ctx, pool, fairPrice, amend.SlippageTolerance, idgeneration.New(deterministicID)); err != nil {
		// couldn't rebase the pool back to its original fair price so the amend is rejected
		if err := e.updateSubAccountBalance(ctx, amend.Party, pool.SubAccount, oldCommitment); err != nil {
			e.log.Panic("could not revert balances are failed rebase", logging.Error(err))
		}
		return err
	}

	// set state to active since if it was closing in reduce-position mode it becomes alive again
	pool.status = types.AMMPoolStatusActive
	e.sendUpdate(ctx, pool)
	return nil
}

func (e *Engine) CancelAMM(
	ctx context.Context,
	cancel *types.CancelAMM,
) (events.Margin, error) {
	pool, ok := e.pools[cancel.Party]
	if !ok {
		return nil, ErrNoPoolMatchingParty
	}

	if cancel.Method == types.AMMPoolCancellationMethodReduceOnly {
		if pos, _ := pool.getPosition(); pos != 0 {
			// pool will now only accept trades that will reduce its position
			pool.status = types.AMMPoolStatusReduceOnly
			e.sendUpdate(ctx, pool)
			return nil, nil
		}
	}

	// either pool has no position or owner wants out right now, so release general balance and
	// get ready for a closeout.
	closeout, err := e.releaseSubAccounts(ctx, pool)
	if err != nil {
		return nil, err
	}

	pool.status = types.AMMPoolStatusCancelled
	e.remove(ctx, cancel.Party)
	return closeout, nil
}

func (e *Engine) StopPool(
	ctx context.Context,
	key string,
) error {
	party, ok := e.subAccounts[key]
	if !ok {
		return ErrNoPoolMatchingParty
	}
	e.remove(ctx, party)
	return nil
}

func (e *Engine) MarketClosing() error { return errors.New("unimplemented") }

func (e *Engine) sendUpdate(ctx context.Context, pool *Pool) {
	e.broker.Send(
		events.NewAMMPoolEvent(
			ctx, pool.party, e.market.GetID(), pool.SubAccount, pool.ID,
			pool.Commitment, pool.Parameters,
			pool.status, types.AMMPoolStatusReasonUnspecified,
		),
	)
}

func (e *Engine) ensureCommitmentAmount(
	_ context.Context,
	commitmentAmount *num.Uint,
) error {
	quantum, _ := e.collateral.GetAssetQuantum(e.market.GetSettlementAsset())
	quantumCommitment := commitmentAmount.ToDecimal().Div(quantum)

	if quantumCommitment.LessThan(e.minCommitmentQuantum.ToDecimal()) {
		return ErrCommitmentTooLow
	}

	return nil
}

// releaseSubAccountGeneralBalance returns the full balance of the sub-accounts general account back to the
// owner of the pool.
func (e *Engine) releaseSubAccounts(ctx context.Context, pool *Pool) (events.Margin, error) {
	var pos events.MarketPosition
	if pp := e.position.GetPositionsByParty(pool.SubAccount); len(pp) > 0 {
		pos = pp[0]
	} else {
		// if a pool is cancelled right after creation it won't have a position yet so we just make an empty one to give
		// to collateral
		pos = positions.NewMarketPosition(pool.SubAccount)
	}

	ledgerMovements, closeout, err := e.collateral.SubAccountRelease(ctx, pool.party, pool.SubAccount, pool.asset, pool.market, pos)
	if err != nil {
		return nil, err
	}

	e.broker.Send(events.NewLedgerMovements(
		ctx, ledgerMovements))
	return closeout, nil
}

func (e *Engine) updateSubAccountBalance(
	ctx context.Context,
	party, subAccount string,
	newCommitment *num.Uint,
) error {
	// first we get the current balance of both the margin, and general subAccount
	subMargin, err := e.collateral.GetPartyMarginAccount(
		e.market.GetID(), subAccount, e.market.GetSettlementAsset())
	if err != nil {
		// by that point the account must exist
		e.log.Panic("no sub margin account", logging.Error(err))
	}
	subGeneral, err := e.collateral.GetPartyGeneralAccount(
		subAccount, e.market.GetSettlementAsset())
	if err != nil {
		// by that point the account must exist
		e.log.Panic("no sub general account", logging.Error(err))
	}

	var (
		currentCommitment = num.Sum(subMargin.Balance, subGeneral.Balance)
		transferType      types.TransferType
		actualAmount      = num.UintZero()
	)

	if currentCommitment.LT(newCommitment) {
		transferType = types.TransferTypeAMMSubAccountLow
		actualAmount.Sub(newCommitment, currentCommitment)
	} else if currentCommitment.GT(newCommitment) {
		transferType = types.TransferTypeAMMSubAccountHigh
		actualAmount.Sub(currentCommitment, newCommitment)
	} else {
		// nothing to do
		return nil
	}

	ledgerMovements, err := e.collateral.SubAccountUpdate(
		ctx, party, subAccount, e.market.GetSettlementAsset(),
		e.market.GetID(), transferType, actualAmount,
	)
	if err != nil {
		return err
	}

	e.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{ledgerMovements}))

	return nil
}

// rebasePool submits an order on behalf of the given pool to pull it fair-price towards the target.
func (e *Engine) rebasePool(ctx context.Context, pool *Pool, target *num.Uint, tol num.Decimal, idgen common.IDGenerator) error {
	if target.LT(pool.lower.low) || target.GT(pool.upper.high) {
		return ErrRebaseTargetOutsideBounds
	}

	// get the pools current fair-price
	fairPrice := pool.BestPrice(nil)
	e.log.Debug("rebasing pool",
		logging.String("id", pool.ID),
		logging.String("fair-price", fairPrice.String()),
		logging.String("target", target.String()),
		logging.String("slippage", tol.String()),
	)
	if fairPrice.EQ(target) {
		return nil
	}

	// calculate slippage as a factor of the mark-price so we can allow for a trades at prices +/- either side of the mark price, depending on side
	slippage, _ := num.UintFromDecimal(target.ToDecimal().Mul(tol))

	// this is the order the pool will submit to rebase itself such that its fair-price is roughly the mark price
	order := &types.OrderSubmission{
		MarketID:    pool.market,
		Price:       num.UintZero(),
		TimeInForce: types.OrderTimeInForceFOK,
		Type:        types.OrderTypeLimit,
		Reference:   fmt.Sprintf("amm-rebase-%s", pool.ID),
	}

	if target.GT(fairPrice) {
		// pool base price is lower than market price, it will need to sell to lower its fair-price
		order.Side = types.SideSell
		order.Price.Sub(target, slippage)
	} else {
		order.Side = types.SideBuy
		order.Price.Add(target, slippage)
	}

	// ask the pool for the volume it would need to shift to get its price to target
	// the order side is the side of the order that will trade with it, so needs to be the opposite
	order.Size = pool.VolumeBetweenPrices(types.OtherSide(order.Side), fairPrice, target)
	if order.Size == 0 {
		// fair-price is so close to target price that the volume to shift it is too small, but thats ok
		return nil
	}

	// need to scale make to market precision here because thats what SubmitOrderWithIDGeneratorAndOrderID expects
	order.Price.Div(order.Price, e.priceFactor)

	e.log.Debug("submitting order to rebase after scale",
		logging.Uint64("size", order.Size),
		logging.String("price", order.Price.String()),
		logging.String("side", order.Side.String()),
	)

	conf, err := e.market.SubmitOrderWithIDGeneratorAndOrderID(ctx, order, pool.SubAccount, idgen, idgen.NextID(), true)
	if err != nil {
		return err
	}

	if conf.Order.Status != types.OrderStatusFilled {
		return ErrRebaseOrderDidNotTrade
	}
	return nil
}

func (e *Engine) GetAMMPools() map[string]common.AMMPool {
	ret := make(map[string]common.AMMPool, len(e.pools))
	for k, v := range e.pools {
		ret[k] = v
	}
	return ret
}

func (e *Engine) add(p *Pool) {
	e.pools[p.party] = p
	e.poolsCpy = append(e.poolsCpy, p)
}

func (e *Engine) remove(ctx context.Context, party string) {
	for i := range e.poolsCpy {
		if e.poolsCpy[i].party == party {
			e.poolsCpy = append(e.poolsCpy[:i], e.poolsCpy[i+1:]...)
			break
		}
	}

	pool := e.pools[party]
	delete(e.pools, party)
	e.sendUpdate(ctx, pool)
}

func DeriveSubAccount(
	party, market, version string,
	index uint64,
) string {
	hash := crypto.Hash([]byte(fmt.Sprintf("%v%v%v%v", version, market, party, index)))
	return hex.EncodeToString(hash)
}