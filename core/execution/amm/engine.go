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

	"golang.org/x/exp/maps"
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
	V1 = "AMMv1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/amm Collateral,Position

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
	SubAccountClosed(ctx context.Context, party, subAccount, asset, market string) ([]*types.LedgerMovement, error)
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

	broker                Broker
	marketActivityTracker *common.MarketActivityTracker

	collateral Collateral
	position   Position
	parties    common.Parties

	marketID string
	assetID  string
	idgen    *idgeneration.IDGenerator

	// gets us from the price in the submission -> price in full asset dp
	priceFactor    num.Decimal
	positionFactor num.Decimal

	// map of party -> pool
	pools    map[string]*Pool
	poolsCpy []*Pool

	// sqrt calculator with cache
	rooter *Sqrter

	// a mapping of all amm-party-ids to the party owning them.
	ammParties map[string]string

	minCommitmentQuantum *num.Uint
	maxCalculationLevels *num.Uint
}

func New(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	marketID string,
	assetID string,
	position Position,
	priceFactor num.Decimal,
	positionFactor num.Decimal,
	marketActivityTracker *common.MarketActivityTracker,
	parties common.Parties,
) *Engine {
	return &Engine{
		log:                   log,
		broker:                broker,
		collateral:            collateral,
		position:              position,
		marketID:              marketID,
		assetID:               assetID,
		marketActivityTracker: marketActivityTracker,
		pools:                 map[string]*Pool{},
		poolsCpy:              []*Pool{},
		ammParties:            map[string]string{},
		minCommitmentQuantum:  num.UintZero(),
		rooter:                &Sqrter{cache: map[string]num.Decimal{}},
		priceFactor:           priceFactor,
		positionFactor:        positionFactor,
		parties:               parties,
	}
}

func NewFromProto(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	marketID string,
	assetID string,
	position Position,
	state *v1.AmmState,
	priceFactor num.Decimal,
	positionFactor num.Decimal,
	marketActivityTracker *common.MarketActivityTracker,
	parties common.Parties,
) (*Engine, error) {
	e := New(log, broker, collateral, marketID, assetID, position, priceFactor, positionFactor, marketActivityTracker, parties)

	for _, v := range state.AmmPartyIds {
		e.ammParties[v.Key] = v.Value
	}

	// TODO consider whether we want the cache in the snapshot, it might be pretty large/slow and I'm not sure what we gain
	for _, v := range state.Sqrter {
		e.rooter.cache[v.Key] = num.MustDecimalFromString(v.Value)
	}

	for _, v := range state.Pools {
		p, err := NewPoolFromProto(e.rooter.sqrt, e.collateral, e.position, v.Pool, v.Party, priceFactor)
		if err != nil {
			return e, err
		}
		e.add(p)
	}

	return e, nil
}

func (e *Engine) IntoProto() *v1.AmmState {
	state := &v1.AmmState{
		Sqrter:      make([]*v1.StringMapEntry, 0, len(e.rooter.cache)),
		AmmPartyIds: make([]*v1.StringMapEntry, 0, len(e.ammParties)),
		Pools:       make([]*v1.PoolMapEntry, 0, len(e.pools)),
	}

	for k, v := range e.rooter.cache {
		state.Sqrter = append(state.Sqrter, &v1.StringMapEntry{
			Key:   k,
			Value: v.String(),
		})
	}
	sort.Slice(state.Sqrter, func(i, j int) bool { return state.Sqrter[i].Key < state.Sqrter[j].Key })

	for k, v := range e.ammParties {
		state.AmmPartyIds = append(state.AmmPartyIds, &v1.StringMapEntry{
			Key:   k,
			Value: v,
		})
	}
	sort.Slice(state.AmmPartyIds, func(i, j int) bool { return state.AmmPartyIds[i].Key < state.AmmPartyIds[j].Key })

	for _, v := range e.poolsCpy {
		state.Pools = append(state.Pools, &v1.PoolMapEntry{
			Party: v.owner,
			Pool:  v.IntoProto(),
		})
	}
	return state
}

func (e *Engine) OnMinCommitmentQuantumUpdate(ctx context.Context, c *num.Uint) {
	e.minCommitmentQuantum = c.Clone()
}

func (e *Engine) OnMaxCalculationLevelsUpdate(ctx context.Context, c *num.Uint) {
	e.maxCalculationLevels = c.Clone()

	for _, p := range e.poolsCpy {
		p.maxCalculationLevels = e.maxCalculationLevels.Clone()
	}
}

// OnMTM is called whenever core does an MTM and is a signal that any pool's that are closing and have 0 position can be fully removed.
func (e *Engine) OnMTM(ctx context.Context) {
	rm := []string{}
	for _, p := range e.poolsCpy {
		if !p.closing() {
			continue
		}
		if pos := p.getPosition(); pos != 0 {
			continue
		}

		// pool is closing and has reached 0 position, we can cancel it now
		if _, err := e.releaseSubAccounts(ctx, p, false); err != nil {
			e.log.Error("unable to release subaccount balance", logging.Error(err))
		}
		p.status = types.AMMPoolStatusCancelled
		rm = append(rm, p.owner)
	}
	for _, party := range rm {
		e.remove(ctx, party)
	}
}

func (e *Engine) OnTick(ctx context.Context, _ time.Time) {
	// seed an id-generator to create IDs for any orders generated in this block
	_, blockHash := vgcontext.TraceIDFromContext(ctx)
	e.idgen = idgeneration.New(blockHash + crypto.HashStrToHex("amm-engine"+e.marketID))

	// any pools that for some reason have zero balance in their accounts will get stopped
	rm := []string{}
	for _, p := range e.poolsCpy {
		if p.getBalance().IsZero() {
			p.status = types.AMMPoolStatusStopped
			rm = append(rm, p.owner)
		}
	}
	for _, party := range rm {
		e.remove(ctx, party)
	}
}

// RemoveDistressed checks if any of the closed out parties are AMM's and if so the AMM is stopped and removed.
func (e *Engine) RemoveDistressed(ctx context.Context, closed []events.MarketPosition) {
	for _, c := range closed {
		owner, ok := e.ammParties[c.Party()]
		if !ok {
			continue
		}
		p, ok := e.pools[owner]
		if !ok {
			e.log.Panic("could not find pool for owner, not possible",
				logging.String("owner", c.Party()),
				logging.String("owner", owner),
			)
		}
		p.status = types.AMMPoolStatusStopped
		e.remove(ctx, owner)
	}
}

// BestPricesAndVolumes returns the best bid/ask and their volumes across all the registered AMM's.
func (e *Engine) BestPricesAndVolumes() (*num.Uint, uint64, *num.Uint, uint64) {
	var bestBid, bestAsk *num.Uint
	var bestBidVolume, bestAskVolume uint64

	for _, pool := range e.poolsCpy {
		// get the pool's current price
		fp := pool.BestPrice(nil)

		// get the volume on the buy side by simulating an incoming sell order
		bid := num.UintZero().Sub(fp, pool.oneTick)
		volume := pool.TradableVolumeInRange(types.SideSell, fp.Clone(), bid)

		if volume != 0 {
			if bestBid == nil || bid.GT(bestBid) {
				bestBid = bid
				bestBidVolume = volume
			} else if bid.EQ(bestBid) {
				bestBidVolume += volume
			}
		}

		// get the volume on the sell side by simulating an incoming buy order
		ask := num.UintZero().Add(fp, pool.oneTick)
		volume = pool.TradableVolumeInRange(types.SideBuy, fp.Clone(), ask)
		if volume != 0 {
			if bestAsk == nil || ask.LT(bestAsk) {
				bestAsk = ask
				bestAskVolume = volume
			} else if ask.EQ(bestAsk) {
				bestAskVolume += volume
			}
		}
	}
	return bestBid, bestBidVolume, bestAsk, bestAskVolume
}

// GetVolumeAtPrice returns the volumes across all registered AMM's that will uncross with with an order at the given price.
// Calling this function with price 1000 and side == sell will return the buy orders that will uncross.
func (e *Engine) GetVolumeAtPrice(price *num.Uint, side types.Side) uint64 {
	vol := uint64(0)
	for _, pool := range e.poolsCpy {
		// get the pool's current price
		fp := pool.BestPrice(nil)
		volume := pool.TradableVolumeInRange(side, fp, price)
		vol += volume
	}
	return vol
}

func (e *Engine) submit(active []*Pool, agg *types.Order, inner, outer *num.Uint) []*types.Order {
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("checking for volume between",
			logging.String("inner", inner.String()),
			logging.String("outer", outer.String()),
		)
	}

	orders := []*types.Order{}
	useActive := make([]*Pool, 0, len(active))
	for _, p := range active {
		p.setEphemeralPosition()

		price := p.BestPrice(agg)
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("best price for pool",
				logging.String("id", p.ID),
				logging.String("best-price", price.String()),
			)
		}

		if agg.Side == types.SideBuy {
			if price.GT(outer) || (agg.Type != types.OrderTypeMarket && price.GT(agg.Price)) {
				// either fair price is out of bounds, or is selling at higher than incoming buy
				continue
			}
		}

		if agg.Side == types.SideSell {
			if price.LT(outer) || (agg.Type != types.OrderTypeMarket && price.LT(agg.Price)) {
				// either fair price is out of bounds, or is buying at lower than incoming sell
				continue
			}
		}
		useActive = append(useActive, p)
	}

	if agg.Side == types.SideSell {
		inner, outer = outer, inner
	}

	// calculate the volume each pool has
	var total uint64
	volumes := []uint64{}
	for _, p := range useActive {
		volume := p.TradableVolumeInRange(agg.Side, inner, outer)
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("volume available to trade",
				logging.Uint64("volume", volume),
				logging.String("id", p.ID),
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
	for i, p := range useActive {
		volume := volumes[i]
		if volume == 0 {
			continue
		}

		// calculate the price the pool wil give for the trading volume
		price := p.PriceForVolume(volume, agg.Side)
		e.log.Info("generated order at price",
			logging.String("price", price.String()),
			logging.Uint64("volume", volume),
			logging.String("id", p.ID),
			logging.String("side", types.OtherSide(agg.Side).String()),
		)

		// construct an order
		o := p.makeOrder(volume, price, types.OtherSide(agg.Side), e.idgen)

		// fill in extra details
		o.CreatedAt = agg.CreatedAt

		orders = append(orders, o)
		p.updateEphemeralPosition(o)

		agg.Remaining -= volume
	}

	return orders
}

// partition takes the given price range and returns which pools have volume in that region, and
// divides that range into sub-levels where AMM boundaries end.
func (e *Engine) partition(agg *types.Order, inner, outer *num.Uint) ([]*Pool, []*num.Uint) {
	active := []*Pool{}
	bounds := map[string]*num.Uint{}

	// cap outer to incoming order price
	if agg.Type != types.OrderTypeMarket {
		switch {
		case outer == nil:
			outer = agg.Price.Clone()
		case agg.Side == types.SideSell && agg.Price.GT(outer):
			outer = agg.Price.Clone()
		case agg.Side == types.SideBuy && agg.Price.LT(outer):
			outer = agg.Price.Clone()
		}
	}

	// switch so that inner < outer to make it easier to reason with
	if agg.Side == types.SideSell {
		inner, outer = outer, inner
	}

	if inner != nil {
		bounds[inner.String()] = inner.Clone()
	}
	if outer != nil {
		bounds[outer.String()] = outer.Clone()
	}

	for _, p := range e.poolsCpy {
		// not active in range if it cannot trade
		if !p.canTrade(agg.Side) {
			continue
		}

		// stop early trying to trade with itself, can happens during auction uncrossing
		if agg.Party == p.AMMParty {
			continue
		}

		// not active in range if its the pool's curves are wholly outside of [inner, outer]
		if (inner != nil && p.upper.high.LT(inner)) || (outer != nil && p.lower.low.GT(outer)) {
			continue
		}

		// pool is active in range add it to the slice
		active = append(active, p)

		// if a pool's upper or lower boundary exists within (inner, outer) then we consider that a sub-level
		boundary := p.upper.high
		if outer == nil || boundary.LT(outer) {
			bounds[boundary.String()] = boundary.Clone()
		}

		boundary = p.lower.low
		if inner == nil || boundary.GT(inner) {
			bounds[boundary.String()] = boundary.Clone()
		}
	}

	// now sort the sub-levels, if the incoming order is a buy we want them ordered ascending so we consider prices in this order:
	// 2000 -> 2100 -> 2200
	//
	// and if its a sell we want them descending so we consider them like:
	// 2000 -> 1900 -> 1800
	levels := maps.Values(bounds)
	sort.Slice(levels,
		func(i, j int) bool {
			if agg.Side == types.SideSell {
				return levels[i].GT(levels[j])
			}
			return levels[i].LT(levels[j])
		},
	)
	return active, levels
}

// SubmitOrder takes an aggressive order and generates matching orders with the registered AMMs such that
// volume is only taken in the interval (inner, outer) where inner and outer are price-levels on the orderbook.
// For example if agg is a buy order inner < outer, and if its a sell outer < inner.
func (e *Engine) SubmitOrder(agg *types.Order, inner, outer *num.Uint) []*types.Order {
	if len(e.pools) == 0 {
		return nil
	}

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("looking for match with order",
			logging.Int("n-pools", len(e.pools)),
			logging.Order(agg),
		)
	}

	// parition the given range into levels where AMM boundaries end
	agg = agg.Clone()
	active, levels := e.partition(agg, inner, outer)

	// submit orders to active pool's between each price level created by any of their high/low boundaries
	orders := []*types.Order{}
	for i := 0; i < len(levels)-1; i++ {
		o := e.submit(active, agg, levels[i], levels[i+1])
		orders = append(orders, o...)

		if agg.Remaining == 0 {
			break
		}
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

// Create takes the definition of an AMM and returns it. It is not considered a participating AMM until Confirm as been called with it.
func (e *Engine) Create(
	ctx context.Context,
	submit *types.SubmitAMM,
	deterministicID string,
	riskFactors *types.RiskFactor,
	scalingFactors *types.ScalingFactors,
	slippage num.Decimal,
) (*Pool, error) {
	idgen := idgeneration.New(deterministicID)
	poolID := idgen.NextID()

	subAccount := DeriveAMMParty(submit.Party, submit.MarketID, V1, 0)
	_, ok := e.pools[submit.Party]
	if ok {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.marketID, subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMStatusReasonPartyAlreadyOwnsAPool,
			),
		)

		return nil, ErrPartyAlreadyOwnAPool(e.marketID)
	}

	if err := e.ensureCommitmentAmount(ctx, submit.Party, subAccount, submit.CommitmentAmount); err != nil {
		reason := types.AMMStatusReasonCannotFillCommitment
		if err == ErrCommitmentTooLow {
			reason = types.AMMStatusReasonCommitmentTooLow
		}
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.marketID, subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, reason,
			),
		)
		return nil, err
	}

	_, _, err := e.collateral.CreatePartyAMMsSubAccounts(ctx, submit.Party, subAccount, e.assetID, submit.MarketID)
	if err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.marketID, subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMStatusReasonUnspecified,
			),
		)

		return nil, err
	}

	pool, err := NewPool(
		poolID,
		subAccount,
		e.assetID,
		submit,
		e.rooter.sqrt,
		e.collateral,
		e.position,
		riskFactors,
		scalingFactors,
		slippage,
		e.priceFactor,
		e.positionFactor,
		e.maxCalculationLevels,
	)
	if err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.marketID, subAccount, poolID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMStatusReasonCommitmentTooLow,
			),
		)

		return nil, err
	}

	e.log.Debug("AMM created",
		logging.String("owner", submit.Party),
		logging.String("poolID", pool.ID),
		logging.String("marketID", e.marketID),
	)
	return pool, nil
}

// Confirm takes an AMM that was created earlier and now commits it to the engine as a functioning pool.
func (e *Engine) Confirm(
	ctx context.Context,
	pool *Pool,
) {
	e.log.Debug("AMM confirmed",
		logging.String("owner", pool.owner),
		logging.String("marketID", e.marketID),
		logging.String("poolID", pool.ID),
	)

	pool.status = types.AMMPoolStatusActive
	pool.maxCalculationLevels = e.maxCalculationLevels

	e.add(pool)
	e.sendUpdate(ctx, pool)
	e.parties.AssignDeriveKey(types.PartyID(pool.owner), pool.AMMParty)
}

// Amend takes the details of an amendment to an AMM and returns a copy of that pool with the updated curves along with the current pool.
// The changes are not taken place in the AMM engine until Confirm is called on the updated pool.
func (e *Engine) Amend(
	ctx context.Context,
	amend *types.AmendAMM,
	riskFactors *types.RiskFactor,
	scalingFactors *types.ScalingFactors,
	slippage num.Decimal,
) (*Pool, *Pool, error) {
	pool, ok := e.pools[amend.Party]
	if !ok {
		return nil, nil, ErrNoPoolMatchingParty
	}

	if amend.CommitmentAmount != nil {
		if err := e.ensureCommitmentAmount(ctx, amend.Party, pool.AMMParty, amend.CommitmentAmount); err != nil {
			return nil, nil, err
		}
	}

	// we need to remove the existing pool from the engine so that when calculating rebasing orders we do not
	// trade with ourselves.
	e.remove(ctx, amend.Party)
	updated, err := pool.Update(amend, riskFactors, scalingFactors, slippage)
	if err != nil {
		return nil, nil, err
	}
	e.log.Debug("AMM amended",
		logging.String("owner", amend.Party),
		logging.String("marketID", e.marketID),
		logging.String("poolID", pool.ID),
	)
	return updated, pool, nil
}

func (e *Engine) CancelAMM(
	ctx context.Context,
	cancel *types.CancelAMM,
) (events.Margin, error) {
	pool, ok := e.pools[cancel.Party]
	if !ok {
		return nil, ErrNoPoolMatchingParty
	}

	if cancel.Method == types.AMMCancellationMethodReduceOnly {
		// pool will now only accept trades that will reduce its position
		pool.status = types.AMMPoolStatusReduceOnly
		e.sendUpdate(ctx, pool)
		return nil, nil
	}

	// either pool has no position or owner wants out right now, so release general balance and
	// get ready for a closeout.
	closeout, err := e.releaseSubAccounts(ctx, pool, false)
	if err != nil {
		return nil, err
	}

	pool.status = types.AMMPoolStatusCancelled
	e.remove(ctx, cancel.Party)
	e.log.Debug("AMM cancelled",
		logging.String("owner", cancel.Party),
		logging.String("poolID", pool.ID),
		logging.String("marketID", e.marketID),
	)
	return closeout, nil
}

func (e *Engine) StopPool(
	ctx context.Context,
	key string,
) error {
	party, ok := e.ammParties[key]
	if !ok {
		return ErrNoPoolMatchingParty
	}
	e.remove(ctx, party)
	return nil
}

// MarketClosing stops all AMM's and returns subaccount balances back to the owning party.
func (e *Engine) MarketClosing(ctx context.Context) error {
	for _, p := range e.poolsCpy {
		if _, err := e.releaseSubAccounts(ctx, p, true); err != nil {
			return err
		}
		p.status = types.AMMPoolStatusStopped
		e.sendUpdate(ctx, p)
		e.marketActivityTracker.RemoveAMMParty(e.assetID, e.marketID, p.AMMParty)
	}
	return nil
}

func (e *Engine) sendUpdate(ctx context.Context, pool *Pool) {
	e.broker.Send(
		events.NewAMMPoolEvent(
			ctx, pool.owner, e.marketID, pool.AMMParty, pool.ID,
			pool.Commitment, pool.Parameters,
			pool.status, types.AMMStatusReasonUnspecified,
		),
	)
}

func (e *Engine) ensureCommitmentAmount(
	_ context.Context,
	party string,
	subAccount string,
	commitmentAmount *num.Uint,
) error {
	quantum, _ := e.collateral.GetAssetQuantum(e.assetID)
	quantumCommitment := commitmentAmount.ToDecimal().Div(quantum)

	if quantumCommitment.LessThan(e.minCommitmentQuantum.ToDecimal()) {
		return ErrCommitmentTooLow
	}

	total := num.UintZero()

	// check they have enough in their accounts, sub-margin + sub-general + general >= commitment
	if a, err := e.collateral.GetPartyMarginAccount(e.marketID, subAccount, e.assetID); err == nil {
		total.Add(total, a.Balance)
	}

	if a, err := e.collateral.GetPartyGeneralAccount(subAccount, e.assetID); err == nil {
		total.Add(total, a.Balance)
	}

	if a, err := e.collateral.GetPartyGeneralAccount(party, e.assetID); err == nil {
		total.Add(total, a.Balance)
	}

	if total.LT(commitmentAmount) {
		return fmt.Errorf("not enough collateral in general account")
	}

	return nil
}

// releaseSubAccountGeneralBalance returns the full balance of the sub-accounts general account back to the
// owner of the pool.
func (e *Engine) releaseSubAccounts(ctx context.Context, pool *Pool, mktClose bool) (events.Margin, error) {
	if mktClose {
		ledgerMovements, err := e.collateral.SubAccountClosed(ctx, pool.owner, pool.AMMParty, pool.asset, pool.market)
		if err != nil {
			return nil, err
		}
		e.broker.Send(events.NewLedgerMovements(ctx, ledgerMovements))
		return nil, nil
	}
	var pos events.MarketPosition
	if pp := e.position.GetPositionsByParty(pool.AMMParty); len(pp) > 0 {
		pos = pp[0]
	} else {
		// if a pool is cancelled right after creation it won't have a position yet so we just make an empty one to give
		// to collateral
		pos = positions.NewMarketPosition(pool.AMMParty)
	}

	ledgerMovements, closeout, err := e.collateral.SubAccountRelease(ctx, pool.owner, pool.AMMParty, pool.asset, pool.market, pos)
	if err != nil {
		return nil, err
	}

	e.broker.Send(events.NewLedgerMovements(
		ctx, ledgerMovements))
	return closeout, nil
}

func (e *Engine) UpdateSubAccountBalance(
	ctx context.Context,
	party, subAccount string,
	newCommitment *num.Uint,
) (*num.Uint, error) {
	// first we get the current balance of both the margin, and general subAccount
	subMargin, err := e.collateral.GetPartyMarginAccount(
		e.marketID, subAccount, e.assetID)
	if err != nil {
		// by that point the account must exist
		e.log.Panic("no sub margin account", logging.Error(err))
	}
	subGeneral, err := e.collateral.GetPartyGeneralAccount(
		subAccount, e.assetID)
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
		transferType = types.TransferTypeAMMLow
		actualAmount.Sub(newCommitment, currentCommitment)
	} else if currentCommitment.GT(newCommitment) {
		transferType = types.TransferTypeAMMHigh
		actualAmount.Sub(currentCommitment, newCommitment)
	} else {
		// nothing to do
		return currentCommitment, nil
	}

	ledgerMovements, err := e.collateral.SubAccountUpdate(
		ctx, party, subAccount, e.assetID,
		e.marketID, transferType, actualAmount,
	)
	if err != nil {
		return nil, err
	}

	e.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{ledgerMovements}))

	return currentCommitment, nil
}

// OrderbookShape expands all registered AMM's into orders between the given prices. If `ammParty` is supplied then just the pool
// with that party id is expanded.
func (e *Engine) OrderbookShape(st, nd *num.Uint, ammParty *string) ([]*types.Order, []*types.Order) {
	if ammParty == nil {
		// no party give, expand all registered
		buys, sells := []*types.Order{}, []*types.Order{}
		for _, p := range e.poolsCpy {
			b, s := p.OrderbookShape(st, nd, e.idgen)
			buys = append(buys, b...)
			sells = append(sells, s...)
		}
		return buys, sells
	}

	// asked to expand just one AMM, lets find it, first amm-party -> owning party
	owner, ok := e.ammParties[*ammParty]
	if !ok {
		return nil, nil
	}

	// now owning party -> pool
	p, ok := e.pools[owner]
	if !ok {
		return nil, nil
	}

	// expand it
	return p.OrderbookShape(st, nd, e.idgen)
}

func (e *Engine) GetAMMPoolsBySubAccount() map[string]common.AMMPool {
	ret := make(map[string]common.AMMPool, len(e.pools))
	for _, v := range e.pools {
		ret[v.AMMParty] = v
	}
	return ret
}

func (e *Engine) GetAllSubAccounts() []string {
	ret := make([]string, 0, len(e.ammParties))
	for _, subAccount := range e.ammParties {
		ret = append(ret, subAccount)
	}
	return ret
}

// GetAMMParty returns the AMM's key given the owners key.
func (e *Engine) GetAMMParty(party string) (string, error) {
	if p, ok := e.pools[party]; ok {
		return p.AMMParty, nil
	}
	return "", ErrNoPoolMatchingParty
}

// IsAMMPartyID returns whether the given key is the key of AMM registered with the engine.
func (e *Engine) IsAMMPartyID(key string) bool {
	_, yes := e.ammParties[key]
	return yes
}

func (e *Engine) add(p *Pool) {
	e.pools[p.owner] = p
	e.poolsCpy = append(e.poolsCpy, p)
	e.ammParties[p.AMMParty] = p.owner
	e.marketActivityTracker.AddAMMSubAccount(e.assetID, e.marketID, p.AMMParty)
}

func (e *Engine) remove(ctx context.Context, party string) {
	for i := range e.poolsCpy {
		if e.poolsCpy[i].owner == party {
			e.poolsCpy = append(e.poolsCpy[:i], e.poolsCpy[i+1:]...)
			break
		}
	}

	pool := e.pools[party]
	delete(e.pools, party)
	delete(e.ammParties, pool.AMMParty)
	e.sendUpdate(ctx, pool)
	e.marketActivityTracker.RemoveAMMParty(e.assetID, e.marketID, pool.AMMParty)
}

func DeriveAMMParty(
	party, market, version string,
	index uint64,
) string {
	hash := crypto.Hash([]byte(fmt.Sprintf("%v%v%v%v", version, market, party, index)))
	return hex.EncodeToString(hash)
}
