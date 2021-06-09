package liquidity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrLiquidityProvisionDoesNotExist  = errors.New("liquidity provision does not exist")
	ErrLiquidityProvisionAlreadyExists = errors.New("liquidity provision already exists")
	ErrCommitmentAmountIsZero          = errors.New("commitment amount is zero")
	ErrEmptyShape                      = errors.New("liquidity provision contains an empty shape")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/liquidity RiskModel,PriceMonitor,IDGen

// Broker - event bus (no mocks needed)
type Broker interface {
	Send(e events.Event)
	SendBatch(evts []events.Event)
}

// RiskModel allows calculation of min/max price range and a probability of trading.
type RiskModel interface {
	ProbabilityOfTrading(currentPrice, yearFraction, orderPrice float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64
	GetProjectionHorizon() float64
}

// PriceMonitor provides the range of valid prices, that is prices that
// wouldn't trade the current trading mode
type PriceMonitor interface {
	GetValidPriceRange() (float64, float64)
}

// IDGen is an id generator for orders.
type IDGen interface {
	SetID(*types.Order)
}

// RepricePeggedOrder reprices a pegged order.
// This function should be injected by the market.
type RepricePeggedOrder func(
	order *types.PeggedOrder, side types.Side,
) (*num.Uint, *types.PeggedOrder, error)

// Engine handles Liquidity provision
type Engine struct {
	marketID       string
	log            *logging.Logger
	broker         Broker
	idGen          IDGen
	suppliedEngine *supplied.Engine

	currentTime             time.Time
	stakeToObligationFactor float64

	// state
	provisions ProvisionsPerParty

	// orders stores all the market orders (except the liquidity orders) explicitly submitted by a given party.
	// indexed as: map of PartyID -> OrderId -> order to easy access
	orders map[string]map[string]*types.Order

	// liquidityOrder stores the orders generated to satisfy the liquidity commitment of a given party.
	// indexed as: map of PartyID -> OrdersID -> order
	liquidityOrders map[string]map[string]*types.Order

	// The list of parties which submitted liquidity submission
	// which still haven't been deployed even once.
	pendings map[string]struct{}

	// the maximum number of liquidity orders to be created on
	// each shape
	maxShapesSize int64

	// this is the max fee that can be specified
	maxFee float64
}

// NewEngine returns a new Liquidity Engine.
func NewEngine(config Config,
	log *logging.Logger,
	broker Broker,
	idGen IDGen,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	marketID string,
) *Engine {
	log.SetLevel(config.Level.Get())
	return &Engine{
		marketID:                marketID,
		log:                     log,
		broker:                  broker,
		idGen:                   idGen,
		suppliedEngine:          supplied.NewEngine(riskModel, priceMonitor),
		stakeToObligationFactor: 1,
		provisions:              map[string]*types.LiquidityProvision{},
		orders:                  map[string]map[string]*types.Order{},
		liquidityOrders:         map[string]map[string]*types.Order{},
		pendings:                map[string]struct{}{},
		maxShapesSize:           100, // set it to the same default than the netparams
		maxFee:                  1.0,
	}
}

// OnChainTimeUpdate updates the internal engine current time
func (e *Engine) OnChainTimeUpdate(ctx context.Context, now time.Time) {
	e.currentTime = now
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v float64) {
	e.suppliedEngine.OnMinProbabilityOfTradingLPOrdersUpdate(v)
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v float64) {
	e.suppliedEngine.OnProbabilityOfTradingTauScalingUpdate(v)
}

// OnSuppliedStakeToObligationFactorUpdate updates the stake factor
func (e *Engine) OnSuppliedStakeToObligationFactorUpdate(v float64) {
	e.stakeToObligationFactor = v
}

func (e *Engine) OnMaximumLiquidityFeeFactorLevelUpdate(f float64) {
	e.maxFee = f
}

func (e *Engine) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	if v < 0 {
		return errors.New("shapes max size cannot be < 0")

	}
	e.maxShapesSize = v
	return nil
}

func (e *Engine) IsPending(party string) bool {
	_, ok := e.pendings[party]
	return ok
}

func (e *Engine) RemovePending(party string) {
	delete(e.pendings, party)
}

func (e *Engine) GetLiquidityOrders(party string) []*types.Order {
	orders := []*types.Order{}
	for _, v := range e.liquidityOrders[party] {
		orders = append(orders, v)
	}
	return orders
}

// GetInactiveParties returns a set of all the parties
// with inactive commitment
func (e *Engine) GetInactiveParties() map[string]struct{} {
	ret := map[string]struct{}{}
	for _, p := range e.provisions {
		if p.Status != types.LiquidityProvision_STATUS_ACTIVE {
			ret[p.PartyId] = struct{}{}
		}
	}
	return ret
}

func (e *Engine) stopLiquidityProvision(
	ctx context.Context, party string, status types.LiquidityProvision_Status) ([]*types.Order, error) {
	lp := e.provisions[party]
	if lp == nil {
		return nil, errors.New("party have no liquidity provision orders")
	}

	lp.Status = status
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	// get the liquidity order to be cancelled
	orders := make([]*types.Order, 0, len(e.liquidityOrders))
	for _, o := range e.liquidityOrders[party] {
		orders = append(orders, o)
	}

	// FIXME(JEREMY): if sorting them is the actual solution
	// review the implementation to write some eventually more efficient
	// way to sort this here and make sure that all orders are always
	// cancelled in the same order
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Id < orders[j].Id
	})

	// now delete all stuff
	delete(e.liquidityOrders, party)
	delete(e.orders, party)
	delete(e.provisions, party)
	delete(e.pendings, party)
	return orders, nil
}

// IsLiquidityProvider returns true if the party hold any liquidity commitmement
func (e *Engine) IsLiquidityProvider(party string) bool {
	_, ok := e.provisions[party]
	return ok
}

// RejectLiquidityProvision removes a parties commitment of liquidity
func (e *Engine) RejectLiquidityProvision(ctx context.Context, party string) error {
	_, err := e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvision_STATUS_REJECTED)
	return err
}

// CancelLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any
func (e *Engine) CancelLiquidityProvision(ctx context.Context, party string) ([]*types.Order, error) {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvision_STATUS_CANCELLED)
}

// StopLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any
func (e *Engine) StopLiquidityProvision(ctx context.Context, party string) ([]*types.Order, error) {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvision_STATUS_STOPPED)
}

// ProvisionsPerParty returns the registered a map of party-id -> LiquidityProvision.
func (e *Engine) ProvisionsPerParty() ProvisionsPerParty {
	return e.provisions
}

func (e *Engine) ValidateLiquidityProvisionSubmission(
	lp *commandspb.LiquidityProvisionSubmission,
	zeroCommitmentIsValid bool,
) (err error) {
	// we check if the commitment is 0 which would mean this is a cancel
	// a cancel does not need validations
	if lp.CommitmentAmount == 0 {
		if zeroCommitmentIsValid {
			return nil
		}
		return ErrCommitmentAmountIsZero
	}

	if fee, err := strconv.ParseFloat(lp.Fee, 64); err != nil || fee < 0 || len(lp.Fee) <= 0 || fee > e.maxFee {
		return errors.New("invalid liquidity provision fee")
	}

	if err := validateShape(lp.Buys, types.Side_SIDE_BUY, e.maxShapesSize); err != nil {
		return err
	}
	return validateShape(lp.Sells, types.Side_SIDE_SELL, e.maxShapesSize)
}

func (e *Engine) rejectLiquidityProvisionSubmission(ctx context.Context, lps *commandspb.LiquidityProvisionSubmission, party, id string) {
	// here we just build a liquidityProvision and set its
	// status to rejected before sending it through the bus
	lp := &types.LiquidityProvision{
		Id:               id,
		Fee:              lps.Fee,
		MarketId:         lps.MarketId,
		PartyId:          party,
		Status:           types.LiquidityProvision_STATUS_REJECTED,
		CreatedAt:        e.currentTime.UnixNano(),
		CommitmentAmount: lps.CommitmentAmount,
		Reference:        lps.Reference,
	}

	lp.Buys = make([]*types.LiquidityOrderReference, 0, len(lps.Buys))
	for _, buy := range lps.Buys {
		lp.Buys = append(lp.Buys, &types.LiquidityOrderReference{
			LiquidityOrder: buy,
		})
	}

	lp.Sells = make([]*types.LiquidityOrderReference, 0, len(lps.Sells))
	for _, sell := range lps.Sells {
		lp.Sells = append(lp.Sells, &types.LiquidityOrderReference{
			LiquidityOrder: sell,
		})
	}

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
}

// SubmitLiquidityProvision handles a new liquidity provision submission.
// It's used to create, update or delete a LiquidityProvision.
// The LiquidityProvision is created if submitted for the first time, updated if a
// previous one was created for the same PartyId or deleted (if exists) when
// the CommitmentAmount is set to 0.
func (e *Engine) SubmitLiquidityProvision(ctx context.Context, lps *commandspb.LiquidityProvisionSubmission, party, id string) error {
	if err := e.ValidateLiquidityProvisionSubmission(lps, false); err != nil {
		e.rejectLiquidityProvisionSubmission(ctx, lps, party, id)
		return err
	}

	if lp := e.LiquidityProvisionByPartyID(party); lp != nil {
		return ErrLiquidityProvisionAlreadyExists
	}

	var (
		now = e.currentTime.UnixNano()
		lp  = &types.LiquidityProvision{
			Id:        id,
			MarketId:  lps.MarketId,
			PartyId:   party,
			CreatedAt: now,
			Fee:       lps.Fee,
			Status:    types.LiquidityProvision_STATUS_REJECTED,
			Reference: lps.Reference,
		}
	)

	// regardless of the final operation (create,update or delete) we finish
	// sending an event.
	defer func() {
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	}()

	e.provisions[party] = lp
	e.orders[party] = map[string]*types.Order{}
	e.liquidityOrders[party] = map[string]*types.Order{}
	e.pendings[party] = struct{}{}
	lp.UpdatedAt = now
	lp.CommitmentAmount = lps.CommitmentAmount
	lp.Status = types.LiquidityProvision_STATUS_PENDING

	e.buildLiquidityProvisionShapesReferences(lp, lps)

	return nil
}

func (e *Engine) buildLiquidityProvisionShapesReferences(
	lp *types.LiquidityProvision,
	lps *commandspb.LiquidityProvisionSubmission,
) {
	// this order is just a stub to send to the id generator,
	// and get an ID assigned per references in the shapes
	order := &types.Order{}
	lp.Buys = make([]*types.LiquidityOrderReference, 0, len(lps.Buys))
	for _, buy := range lps.Buys {
		e.idGen.SetID(order)
		lp.Buys = append(lp.Buys, &types.LiquidityOrderReference{
			OrderId:        order.Id,
			LiquidityOrder: buy,
		})
	}

	lp.Sells = make([]*types.LiquidityOrderReference, 0, len(lps.Sells))
	for _, sell := range lps.Sells {
		e.idGen.SetID(order)
		lp.Sells = append(lp.Sells, &types.LiquidityOrderReference{
			OrderId:        order.Id,
			LiquidityOrder: sell,
		})
	}
}

// LiquidityProvisionByPartyID returns the LP associated to a Party if any.
// If not, it returns nil.
func (e *Engine) LiquidityProvisionByPartyID(partyID string) *types.LiquidityProvision {
	return e.provisions[partyID]
}

func (e *Engine) updatePartyOrders(partyID string, orders []*types.Order) {
	// These maps are created by SubmitLiquidityProvision
	m := e.orders[partyID]
	lm := e.liquidityOrders[partyID]
	if lm == nil {
		return
	}

	for _, order := range orders {
		// skip if it's a liquidity order
		if len(order.LiquidityProvisionId) > 0 {
			continue
		}
		if _, ok := lm[order.Id]; ok {
			continue
		}

		// Remove
		if order.Status != types.Order_STATUS_ACTIVE {
			delete(m, order.Id)
			continue
		}

		// Create or Modify
		m[order.Id] = order
	}
}

// IsLiquidityOrder checks to see if a given order is part of the LP orders for a given party
func (e *Engine) IsLiquidityOrder(party, order string) bool {
	pos, ok := e.liquidityOrders[party]
	if !ok {
		return false
	}
	_, ok = pos[order]
	return ok
}

// CreateInitialOrders returns two slices of orders, one for orders to be
// created and the other for orders to be updated.
func (e *Engine) CreateInitialOrders(
	ctx context.Context,
	bestBidPrice, bestAskPrice *num.Uint,
	party string,
	orders []*types.Order,
	repriceFn RepricePeggedOrder,
) ([]*types.Order, error) {
	// update our internal orders
	e.updatePartyOrders(party, orders)

	// ignoring amends as there won't be any since we kill all the orders first
	creates, _, err := e.createOrUpdateForParty(ctx,
		bestBidPrice, bestAskPrice, party, repriceFn)
	return creates, err
}

// Update gets the order changes.
// It keeps track of all LP orders.
func (e *Engine) Update(
	ctx context.Context,
	bestBidPrice, bestAskPrice *num.Uint,
	repriceFn RepricePeggedOrder,
	orders []*types.Order,
) ([]*types.Order, []*ToCancel, error) {
	var (
		newOrders []*types.Order
		toCancel  []*ToCancel
	)

	// first update internal state of LP orders
	for _, po := range Orders(orders).ByParty() {
		if !e.IsLiquidityProvider(po.Party) {
			continue
		}

		// update our internal orders
		e.updatePartyOrders(po.Party, po.Orders)
	}

	for _, lp := range e.provisions.Slice() {
		creates, cancels, err := e.createOrUpdateForParty(ctx, bestBidPrice, bestAskPrice, lp.PartyId, repriceFn)
		if err != nil {
			return nil, nil, err
		}
		newOrders = append(newOrders, creates...)
		if !cancels.Empty() {
			toCancel = append(toCancel, cancels)
		}
	}
	return newOrders, toCancel, nil
}

// CalculateSuppliedStake returns the sum of commitment amounts from all the liquidity providers
func (e *Engine) CalculateSuppliedStake() uint64 {
	var ss uint64 = 0
	for _, v := range e.provisions {
		ss += v.CommitmentAmount
	}
	return ss
}

func (e *Engine) createOrUpdateForParty(
	ctx context.Context,
	bestBidPrice, bestAskPrice *num.Uint,
	party string,
	repriceFn RepricePeggedOrder,
) ([]*types.Order, *ToCancel, error) {
	lp := e.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return nil, nil, nil
	}

	var (
		obligation     = float64(lp.CommitmentAmount) * e.stakeToObligationFactor
		buysShape      = make([]*supplied.LiquidityOrder, 0, len(lp.Buys))
		sellsShape     = make([]*supplied.LiquidityOrder, 0, len(lp.Sells))
		repriceFailure bool
	)

	for _, buy := range lp.Buys {
		pegged := &types.PeggedOrder{
			Reference: buy.LiquidityOrder.Reference,
			Offset:    buy.LiquidityOrder.Offset,
		}
		order := &supplied.LiquidityOrder{
			OrderID:    buy.OrderId,
			Proportion: uint64(buy.LiquidityOrder.Proportion),
		}
		if price, peggedO, err := repriceFn(pegged, types.Side_SIDE_BUY); err != nil {
			e.log.Debug("Building Buy Shape", logging.Error(err))
			repriceFailure = true
		} else {
			order.Price = price
			order.Peg = peggedO
		}
		buysShape = append(buysShape, order)
	}

	for _, sell := range lp.Sells {
		pegged := &types.PeggedOrder{
			Reference: sell.LiquidityOrder.Reference,
			Offset:    sell.LiquidityOrder.Offset,
		}
		order := &supplied.LiquidityOrder{
			OrderID:    sell.OrderId,
			Proportion: uint64(sell.LiquidityOrder.Proportion),
		}
		if price, peggedO, err := repriceFn(pegged, types.Side_SIDE_SELL); err != nil {
			e.log.Debug("Building Sell Shape", logging.Error(err))
			repriceFailure = true
		} else {
			order.Price = price
			order.Peg = peggedO
		}
		sellsShape = append(sellsShape, order)
	}

	var (
		needsCreateBuys, needsCreateSells []*types.Order
		needsUpdateBuys, needsUpdateSells *ToCancel
	)

	if repriceFailure {
		needsUpdateBuys = e.undeployOrdersFromShape(
			party, buysShape, types.Side_SIDE_BUY)
		needsUpdateSells = e.undeployOrdersFromShape(
			party, sellsShape, types.Side_SIDE_SELL)

		// set to undeployed if active basically as
		// we want to keep it pending until it deployed for the first time
		if lp.Status != types.LiquidityProvision_STATUS_UNDEPLOYED &&
			lp.Status != types.LiquidityProvision_STATUS_PENDING {
			lp.Status = types.LiquidityProvision_STATUS_UNDEPLOYED
		}
	} else {
		// Create a slice shaped copy of the orders
		orders := make([]*types.Order, 0, len(e.orders[party]))
		for _, order := range e.orders[party] {
			orders = append(orders, order)
		}

		if err := e.suppliedEngine.CalculateLiquidityImpliedVolumes(
			bestBidPrice, bestAskPrice,
			obligation,
			orders,
			buysShape, sellsShape,
		); err != nil {
			return nil, nil, err
		}

		needsCreateBuys, needsUpdateBuys = e.createOrdersFromShape(
			party, buysShape, types.Side_SIDE_BUY)
		needsCreateSells, needsUpdateSells = e.createOrdersFromShape(
			party, sellsShape, types.Side_SIDE_SELL)

		lp.Status = types.LiquidityProvision_STATUS_ACTIVE
	}

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	return append(needsCreateBuys, needsCreateSells...),
		needsUpdateBuys.Merge(needsUpdateSells),
		nil
}

func (e *Engine) buildOrder(side types.Side, pegged *types.PeggedOrder, price *num.Uint, partyID, marketID string, size uint64, ref string, lpID string) *types.Order {
	order := &types.Order{
		MarketId:             marketID,
		Side:                 side,
		PeggedOrder:          pegged,
		Price:                price,
		PartyId:              partyID,
		Size:                 size,
		Remaining:            size,
		Type:                 types.Order_TYPE_LIMIT,
		TimeInForce:          types.Order_TIME_IN_FORCE_GTC,
		Reference:            ref,
		LiquidityProvisionId: lpID,
	}
	return order.Create(e.currentTime)
}

func (e *Engine) undeployOrdersFromShape(
	party string, supplied []*supplied.LiquidityOrder, side types.Side,
) *ToCancel {
	lm, ok := e.liquidityOrders[party]
	if !ok {
		lm = map[string]*types.Order{}
		e.liquidityOrders[party] = lm
		if _, ok := e.orders[party]; !ok {
			e.orders[party] = map[string]*types.Order{}
		}
	}

	var (
		toCancel = &ToCancel{
			Party: party,
		}
		lp = e.LiquidityProvisionByPartyID(party)
	)

	for i, o := range supplied {
		var (
			order = lm[o.OrderID]
			ref   *types.LiquidityOrderReference
		)
		if side == types.Side_SIDE_BUY {
			ref = lp.Buys[i]
		} else {
			ref = lp.Sells[i]
		}

		if order != nil {
			// only amend if order remaining > 0
			// if not the market already took care in cleaning
			// up everything
			if order.Remaining != 0 {
				toCancel.Add(order.Id)
			}

			// then we can delete the order from our mapping
			delete(lm, ref.OrderId)
		}
	}

	return toCancel
}

func (e *Engine) createOrdersFromShape(
	party string, supplied []*supplied.LiquidityOrder, side types.Side,
) ([]*types.Order, *ToCancel) {
	lm, ok := e.liquidityOrders[party]
	if !ok {
		lm = map[string]*types.Order{}
		e.liquidityOrders[party] = lm
		if _, ok := e.orders[party]; !ok {
			e.orders[party] = map[string]*types.Order{}
		}
	}
	lp := e.LiquidityProvisionByPartyID(party)

	var (
		newOrders []*types.Order
		toCancel  = &ToCancel{
			Party: party,
		}
	)

	for i, o := range supplied {
		order := lm[o.OrderID]
		var ref *types.LiquidityOrderReference
		if side == types.Side_SIDE_BUY {
			ref = lp.Buys[i]
		} else {
			ref = lp.Sells[i]
		}

		if order != nil && (order.HasTraded() || order.Size != o.LiquidityImpliedVolume) {
			// we always remove the order from our store, and add it to the amendment

			// only amend if order remaining > 0
			// if not the market already took care in cleaning
			// up everything
			if order.Remaining != 0 {
				toCancel.Add(order.Id)
			}

			// then we can delete the order from our mapping
			delete(lm, ref.OrderId)
		}

		// We either don't need this order anymore or
		// we have just nothing to do about it.
		if o.LiquidityImpliedVolume == 0 ||
			// we check if the order was not nil, which mean we alread had a deployed order
			// if the order as not traded, and the size haven't changed, then we have nothing
			// to do about it. If the size has changed, then we will want to recreate one.
			(order != nil && (!order.HasTraded() && order.Size == o.LiquidityImpliedVolume)) ||
			// we check o.Price == 0 just to make sure we are able to price
			// the order, in which case the size will have been calculated
			// properly by the engine.
			o.Price == num.NewUint(0) {
			continue
		}

		// At this point the order will either already exists
		// or not, and we'll want to re-create
		// then we create the new order
		// p := &types.PeggedOrder{
		// 	Reference: ref.LiquidityOrder.Reference,
		// 	Offset:    ref.LiquidityOrder.Offset,
		// }
		order = e.buildOrder(side, o.Peg, o.Price, party, e.marketID, o.LiquidityImpliedVolume, lp.Reference, lp.Id)
		order.Id = ref.OrderId
		newOrders = append(newOrders, order)
		lm[order.Id] = order
		ref.OrderId = order.Id
	}

	return newOrders, toCancel
}

func validateShape(sh []*types.LiquidityOrder, side types.Side, maxSize int64) error {
	if len(sh) <= 0 {
		return fmt.Errorf("empty %v shape", side)
	}
	if len(sh) > int(maxSize) {
		return fmt.Errorf("%v shape size exceed max (%v)", side, maxSize)
	}
	for _, lo := range sh {
		if lo.Reference == types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			// We must specify a valid reference
			return errors.New("order in shape without reference")
		}
		if lo.Proportion == 0 {
			return errors.New("order in shape without a proportion")
		}

		if side == types.Side_SIDE_BUY {
			switch lo.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				return errors.New("order in buy side shape with best ask price reference")
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				if lo.Offset > 0 {
					return errors.New("order in buy side shape offset must be <= 0")
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if lo.Offset >= 0 {
					return errors.New("order in buy side shape offset must be < 0")
				}
			}
		} else {
			switch lo.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				if lo.Offset < 0 {
					return errors.New("order in sell shape offset must be >= 0")
				}
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				return errors.New("order in buy side shape with best ask price reference")
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if lo.Offset <= 0 {
					return errors.New("order in sell shape offset must be > 0")
				}
			}
		}
	}
	return nil
}
