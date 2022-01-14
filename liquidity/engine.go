package liquidity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/logging"
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

// Broker - event bus (no mocks needed).
type Broker interface {
	Send(e events.Event)
	SendBatch(evts []events.Event)
}

// RiskModel allows calculation of min/max price range and a probability of trading.
type RiskModel interface {
	ProbabilityOfTrading(currentPrice, orderPrice *num.Uint, minPrice, maxPrice num.Decimal, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// PriceMonitor provides the range of valid prices, that is prices that
// wouldn't trade the current trading mode.
type PriceMonitor interface {
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
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

// Engine handles Liquidity provision.
type Engine struct {
	marketID       string
	log            *logging.Logger
	broker         Broker
	idGen          IDGen
	suppliedEngine *supplied.Engine

	currentTime             time.Time
	stakeToObligationFactor num.Decimal

	// state
	provisions *SnapshotableProvisionsPerParty

	// orders stores all the market orders (except the liquidity orders) explicitly submitted by a given party.
	// indexed as: map of PartyID -> OrderId -> order to easy access
	orders *SnapshotablePartiesOrders

	// liquidityOrder stores the orders generated to satisfy the liquidity commitment of a given party.
	// indexed as: map of PartyID -> OrdersID -> order
	liquidityOrders *SnapshotablePartiesOrders

	// The list of parties which submitted liquidity submission
	// which still haven't been deployed even once.
	pendings *SnapshotablePendingProvisions

	// the maximum number of liquidity orders to be created on
	// each shape
	maxShapesSize int64

	// this is the max fee that can be specified
	maxFee num.Decimal
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
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	return &Engine{
		marketID:       marketID,
		log:            log,
		broker:         broker,
		idGen:          idGen,
		suppliedEngine: supplied.NewEngine(riskModel, priceMonitor, marketID),

		// parameters
		stakeToObligationFactor: num.DecimalFromInt64(1),
		maxShapesSize:           100, // set it to the same default than the netparams
		maxFee:                  num.DecimalFromInt64(1),
		// provisions related state
		provisions: newSnapshotableProvisionsPerParty(),
		pendings:   newSnapshotablePendingProvisions(),
		// orders related state
		orders:          newSnapshotablePartiesOrders(),
		liquidityOrders: newSnapshotablePartiesOrders(),
	}
}

// OnChainTimeUpdate updates the internal engine current time.
func (e *Engine) OnChainTimeUpdate(_ context.Context, now time.Time) {
	e.currentTime = now
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v num.Decimal) {
	e.suppliedEngine.OnMinProbabilityOfTradingLPOrdersUpdate(v)
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v num.Decimal) {
	e.suppliedEngine.OnProbabilityOfTradingTauScalingUpdate(v)
}

// OnSuppliedStakeToObligationFactorUpdate updates the stake factor.
func (e *Engine) OnSuppliedStakeToObligationFactorUpdate(v num.Decimal) {
	e.stakeToObligationFactor = v
}

func (e *Engine) OnMaximumLiquidityFeeFactorLevelUpdate(f num.Decimal) {
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
	return e.pendings.Exists(party)
}

func (e *Engine) RemovePending(party string) {
	e.pendings.Delete(party)
}

func (e *Engine) GetAllLiquidityOrders() []*types.Order {
	orders := []*types.Order{}
	for _, v := range e.liquidityOrders.m {
		for _, o := range v {
			if o.Status == types.OrderStatusActive {
				orders = append(orders, o)
			}
		}
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})

	return orders
}

func (e *Engine) GetLiquidityOrders(party string) []*types.Order {
	orders := []*types.Order{}
	porders, ok := e.liquidityOrders.GetForParty(party)
	if !ok {
		return nil
	}
	for _, v := range porders {
		orders = append(orders, v)
	}
	return orders
}

// GetInactiveParties returns a set of all the parties
// with inactive commitment.
func (e *Engine) GetInactiveParties() map[string]struct{} {
	ret := map[string]struct{}{}
	for _, p := range e.provisions.ProvisionsPerParty {
		if p.Status != types.LiquidityProvisionStatusActive {
			ret[p.Party] = struct{}{}
		}
	}
	return ret
}

func (e *Engine) stopLiquidityProvision(
	ctx context.Context, party string, status types.LiquidityProvisionStatus,
) ([]*types.Order, error) {
	lp, ok := e.provisions.Get(party)
	if !ok {
		return nil, errors.New("party have no liquidity provision orders")
	}

	lp.Status = status
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	// get the liquidity order to be cancelled
	lorders, _ := e.liquidityOrders.GetForParty(party)
	orders := make([]*types.Order, 0, len(lorders))
	for _, o := range lorders {
		orders = append(orders, o)
	}

	// FIXME(JEREMY): if sorting them is the actual solution
	// review the implementation to write some eventually more efficient
	// way to sort this here and make sure that all orders are always
	// cancelled in the same order
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})

	// now delete all stuff
	e.liquidityOrders.DeleteParty(party)
	e.orders.DeleteParty(party)
	e.provisions.Delete(party)
	e.pendings.Delete(party)
	return orders, nil
}

// IsLiquidityProvider returns true if the party hold any liquidity commitmement.
func (e *Engine) IsLiquidityProvider(party string) bool {
	_, ok := e.provisions.Get(party)
	return ok
}

// RejectLiquidityProvision removes a parties commitment of liquidity.
func (e *Engine) RejectLiquidityProvision(ctx context.Context, party string) error {
	_, err := e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusRejected)
	return err
}

// CancelLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any.
func (e *Engine) CancelLiquidityProvision(ctx context.Context, party string) ([]*types.Order, error) {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusCancelled)
}

// StopLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any.
func (e *Engine) StopLiquidityProvision(ctx context.Context, party string) ([]*types.Order, error) {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusStopped)
}

// ProvisionsPerParty returns the registered a map of party-id -> LiquidityProvision.
func (e *Engine) ProvisionsPerParty() ProvisionsPerParty {
	return e.provisions.ProvisionsPerParty
}

func (e *Engine) ValidateLiquidityProvisionSubmission(
	lp *types.LiquidityProvisionSubmission,
	zeroCommitmentIsValid bool,
) (err error) {
	// we check if the commitment is 0 which would mean this is a cancel
	// a cancel does not need validations
	if lp.CommitmentAmount.IsZero() {
		if zeroCommitmentIsValid {
			return nil
		}
		return ErrCommitmentAmountIsZero
	}

	// not sure how to check for a missing fee, 0 could be valid
	// then again, that validation should've happened before reaching this point
	// if fee, err := strconv.ParseFloat(lp.Fee, 64); err != nil || fee < 0 || len(lp.Fee) <= 0 || fee > e.maxFee {
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	if err := validateShape(lp.Buys, types.SideBuy, e.maxShapesSize); err != nil {
		return err
	}
	return validateShape(lp.Sells, types.SideSell, e.maxShapesSize)
}

func (e *Engine) ValidateLiquidityProvisionAmendment(lp *types.LiquidityProvisionAmendment) (err error) {
	if lp.Fee.IsZero() && !lp.ContainsOrders() && (lp.CommitmentAmount == nil || lp.CommitmentAmount.IsZero()) {
		return errors.New("empty liquidity provision amendment content")
	}

	// If orders fee is provided, we need it to be valid
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	// If orders shapes are provided, we need them to be valid
	if len(lp.Buys) > 0 {
		if err := validateShape(lp.Buys, types.SideBuy, e.maxShapesSize); err != nil {
			return err
		}
	}
	if len(lp.Sells) > 0 {
		if err := validateShape(lp.Sells, types.SideSell, e.maxShapesSize); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) rejectLiquidityProvisionSubmission(ctx context.Context, lps *types.LiquidityProvisionSubmission, party, id string) {
	// here we just build a liquidityProvision and set its
	// status to rejected before sending it through the bus
	lp := &types.LiquidityProvision{
		ID:               id,
		Fee:              lps.Fee,
		MarketID:         lps.MarketID,
		Party:            party,
		Status:           types.LiquidityProvisionStatusRejected,
		CreatedAt:        e.currentTime.UnixNano(),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
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
func (e *Engine) SubmitLiquidityProvision(ctx context.Context, lps *types.LiquidityProvisionSubmission, party, id string) error {
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
			ID:        id,
			MarketID:  lps.MarketID,
			Party:     party,
			CreatedAt: now,
			Fee:       lps.Fee,
			Status:    types.LiquidityProvisionStatusRejected,
			Reference: lps.Reference,
		}
	)

	// regardless of the final operation (create,update or delete) we finish
	// sending an event.
	defer func() {
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	}()

	e.provisions.Set(party, lp)
	e.orders.ResetForParty(party)
	e.liquidityOrders.ResetForParty(party)
	e.pendings.Add(party)
	lp.UpdatedAt = now
	lp.CommitmentAmount = lps.CommitmentAmount
	lp.Status = types.LiquidityProvisionStatusPending

	e.buildLiquidityProvisionShapesReferences(lp, lps.Buys, lps.Sells)

	return nil
}

func (e *Engine) buildLiquidityProvisionShapesReferences(
	lp *types.LiquidityProvision,
	buys []*types.LiquidityOrder,
	sells []*types.LiquidityOrder,
) {
	// this order is just a stub to send to the id generator,
	// and get an ID assigned per references in the shapes
	order := &types.Order{}
	lp.Buys = make([]*types.LiquidityOrderReference, 0, len(buys))
	for _, buy := range buys {
		e.idGen.SetID(order)
		lp.Buys = append(lp.Buys, &types.LiquidityOrderReference{
			OrderID:        order.ID,
			LiquidityOrder: buy,
		})
	}

	lp.Sells = make([]*types.LiquidityOrderReference, 0, len(sells))
	for _, sell := range sells {
		e.idGen.SetID(order)
		lp.Sells = append(lp.Sells, &types.LiquidityOrderReference{
			OrderID:        order.ID,
			LiquidityOrder: sell,
		})
	}
}

// LiquidityProvisionByPartyID returns the LP associated to a Party if any.
// If not, it returns nil.
func (e *Engine) LiquidityProvisionByPartyID(partyID string) *types.LiquidityProvision {
	lp, _ := e.provisions.Get(partyID)
	return lp
}

func (e *Engine) updatePartyOrders(partyID string, orders []*types.Order) {
	// These maps are created by SubmitLiquidityProvision
	_, ok := e.orders.GetForParty(partyID)
	if !ok {
		return
	}
	lm, ok := e.liquidityOrders.GetForParty(partyID)
	if !ok {
		return
	}

	for _, order := range orders {
		// skip if it's a liquidity order
		if len(order.LiquidityProvisionID) > 0 {
			continue
		}
		if _, ok := lm[order.ID]; ok {
			continue
		}

		// Remove
		if order.Status != types.OrderStatusActive {
			e.orders.Delete(order.Party, order.ID)
			continue
		}

		// Create or Modify
		e.orders.Add(order.Party, order)
	}
}

// IsLiquidityOrder checks to see if a given order is part of the LP orders for a given party.
func (e *Engine) IsLiquidityOrder(party, order string) bool {
	_, ok := e.liquidityOrders.Get(party, order)
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
		creates, cancels, err := e.createOrUpdateForParty(ctx, bestBidPrice.Clone(), bestAskPrice.Clone(), lp.Party, repriceFn)
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

// CalculateSuppliedStake returns the sum of commitment amounts from all the liquidity providers.
func (e *Engine) CalculateSuppliedStake() *num.Uint {
	ss := num.Zero()
	for _, v := range e.provisions.ProvisionsPerParty {
		ss.AddSum(v.CommitmentAmount)
	}
	return ss
}

func (e *Engine) createOrUpdateForParty(
	ctx context.Context,
	bestBidPrice, bestAskPrice *num.Uint,
	party string,
	repriceFn RepricePeggedOrder,
) (ordres []*types.Order, _ *ToCancel, errr error) {
	lp := e.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return nil, nil, nil
	}

	var (
		obligation, _ = num.UintFromDecimal(lp.CommitmentAmount.ToDecimal().Mul(e.stakeToObligationFactor).Round(0))
		// Fix this after we update the commentamount to use Uint TODO UINT
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
			OrderID:    buy.OrderID,
			Proportion: uint64(buy.LiquidityOrder.Proportion),
		}
		if price, peggedO, err := repriceFn(pegged, types.SideBuy); err != nil {
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
			OrderID:    sell.OrderID,
			Proportion: uint64(sell.LiquidityOrder.Proportion),
		}
		if price, peggedO, err := repriceFn(pegged, types.SideSell); err != nil {
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
			party, buysShape, types.SideBuy)
		needsUpdateSells = e.undeployOrdersFromShape(
			party, sellsShape, types.SideSell)

		// set to undeployed if active basically as
		// we want to keep it pending until it deployed for the first time
		if lp.Status != types.LiquidityProvisionStatusUndeployed &&
			lp.Status != types.LiquidityProvisionStatusPending {
			lp.Status = types.LiquidityProvisionStatusUndeployed
		}
	} else {
		// Create a slice shaped copy of the orders
		orders := make([]*types.Order, 0, len(e.orders.m[party]))
		for _, order := range e.orders.m[party] {
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
			party, buysShape, types.SideBuy)
		needsCreateSells, needsUpdateSells = e.createOrdersFromShape(
			party, sellsShape, types.SideSell)

		lp.Status = types.LiquidityProvisionStatusActive
	}

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	return append(needsCreateBuys, needsCreateSells...),
		needsUpdateBuys.Merge(needsUpdateSells),
		nil
}

func (e *Engine) buildOrder(side types.Side, price *num.Uint, partyID, marketID string, size uint64, ref string, lpID string) *types.Order {
	order := &types.Order{
		MarketID:             marketID,
		Side:                 side,
		Price:                price.Clone(),
		Party:                partyID,
		Size:                 size,
		Remaining:            size,
		Type:                 types.OrderTypeLimit,
		TimeInForce:          types.OrderTimeInForceGTC,
		Reference:            ref,
		LiquidityProvisionID: lpID,
	}
	return order.Create(e.currentTime)
}

func (e *Engine) undeployOrdersFromShape(
	party string, supplied []*supplied.LiquidityOrder, side types.Side,
) *ToCancel {
	lm, ok := e.liquidityOrders.GetForParty(party)
	if !ok {
		e.liquidityOrders.ResetForParty(party)
		if _, ok := e.orders.GetForParty(party); !ok {
			e.orders.ResetForParty(party)
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
		if side == types.SideBuy {
			ref = lp.Buys[i]
		} else {
			ref = lp.Sells[i]
		}

		if order != nil {
			// only amend if order remaining > 0
			// if not the market already took care in cleaning
			// up everything
			if order.Remaining != 0 {
				toCancel.Add(order.ID)
			}

			// then we can delete the order from our mapping
			e.liquidityOrders.Delete(order.Party, order.ID)
			delete(lm, ref.OrderID)
		}
	}

	return toCancel
}

func (e *Engine) createOrdersFromShape(
	party string, supplied []*supplied.LiquidityOrder, side types.Side,
) ([]*types.Order, *ToCancel) {
	lm, ok := e.liquidityOrders.GetForParty(party)
	if !ok {
		e.liquidityOrders.ResetForParty(party)
		if _, ok := e.orders.GetForParty(party); !ok {
			e.orders.ResetForParty(party)
		}
	}
	lp := e.LiquidityProvisionByPartyID(party)

	var (
		newOrders = make([]*types.Order, 0, len(supplied))
		toCancel  = &ToCancel{
			Party: party,
		}
	)

	for i, o := range supplied {
		order := lm[o.OrderID]
		var ref *types.LiquidityOrderReference
		if side == types.SideBuy {
			ref = lp.Buys[i]
		} else {
			ref = lp.Sells[i]
		}

		if order != nil && (order.HasTraded() || order.Size != o.LiquidityImpliedVolume || order.Price.NEQ(o.Price)) {
			// we always remove the order from our store, and add it to the amendment

			// only amend if order remaining > 0
			// if not the market already took care in cleaning
			// up everything
			if order.Remaining != 0 {
				toCancel.Add(order.ID)
			}

			// then we can delete the order from our mapping
			e.liquidityOrders.Delete(order.Party, order.ID)
		}

		// We either don't need this order anymore or
		// we have just nothing to do about it.
		if o.LiquidityImpliedVolume == 0 ||
			// we check if the order was not nil, which mean we already had a deployed order
			// if the order as not traded, and the size haven't changed, then we have nothing
			// to do about it. If the size has changed, then we will want to recreate one.
			(order != nil && (!order.HasTraded() && order.Size == o.LiquidityImpliedVolume)) ||
			// we check o.Price == 0 just to make sure we are able to price
			// the order, in which case the size will have been calculated
			// properly by the engine.
			o.Price.IsZero() {
			continue
		}

		// At this point the order will either already exists
		// or not, and we'll want to re-create
		// then we create the new order
		// p := &types.PeggedOrder{
		// 	Reference: ref.LiquidityOrder.Reference,
		// 	Offset:    ref.LiquidityOrder.Offset,
		// }
		order = e.buildOrder(side, o.Price, party, e.marketID, o.LiquidityImpliedVolume, lp.Reference, lp.ID)
		order.ID = ref.OrderID
		newOrders = append(newOrders, order)
		e.liquidityOrders.Add(order.Party, order)
		ref.OrderID = order.ID
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
		if lo.Reference == types.PeggedReferenceUnspecified {
			// We must specify a valid reference
			return errors.New("order in shape without reference")
		}
		if lo.Proportion == 0 {
			return errors.New("order in shape without a proportion")
		}

		if side == types.SideBuy {
			switch lo.Reference {
			case types.PeggedReferenceBestAsk:
				return errors.New("order in buy side shape with best ask price reference")
			case types.PeggedReferenceBestBid:
				if lo.Offset > 0 {
					return errors.New("order in buy side shape offset must be <= 0")
				}
			case types.PeggedReferenceMid:
				if lo.Offset >= 0 {
					return errors.New("order in buy side shape offset must be < 0")
				}
			}
		} else {
			switch lo.Reference {
			case types.PeggedReferenceBestAsk:
				if lo.Offset < 0 {
					return errors.New("order in sell shape offset must be >= 0")
				}
			case types.PeggedReferenceBestBid:
				return errors.New("order in buy side shape with best ask price reference")
			case types.PeggedReferenceMid:
				if lo.Offset <= 0 {
					return errors.New("order in sell shape offset must be > 0")
				}
			}
		}
	}
	return nil
}
