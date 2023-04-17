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
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity/supplied"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

var (
	ErrLiquidityProvisionDoesNotExist  = errors.New("liquidity provision does not exist")
	ErrLiquidityProvisionAlreadyExists = errors.New("liquidity provision already exists")
	ErrCommitmentAmountIsZero          = errors.New("commitment amount is zero")
	ErrEmptyShape                      = errors.New("liquidity provision contains an empty shape")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/liquidity RiskModel,PriceMonitor,IDGen

// Broker - event bus (no mocks needed).
type Broker interface {
	Send(e events.Event)
	SendBatch(evts []events.Event)
}

// TimeService provide the time of the vega node using the tm time.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// RiskModel allows calculation of min/max price range and a probability of trading.
type RiskModel interface {
	ProbabilityOfTrading(currentPrice, orderPrice num.Decimal, minPrice, maxPrice num.Decimal, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// PriceMonitor provides the range of valid prices, that is prices that
// wouldn't trade the current trading mode.
type PriceMonitor interface {
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
}

// IDGen is an id generator for orders.
type IDGen interface {
	NextID() string
}

type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
}

// RepriceOrder reprices a pegged order.
// This function should be injected by the market.
type RepriceOrder func(
	side types.Side, reference types.PeggedReference, offset *num.Uint,
) (*num.Uint, error)

// Engine handles Liquidity provision.
type Engine struct {
	marketID       string
	log            *logging.Logger
	timeService    TimeService
	broker         Broker
	suppliedEngine *supplied.Engine
	orderBook      OrderBook

	stakeToObligationFactor num.Decimal

	// state
	provisions *SnapshotableProvisionsPerParty

	// The list of parties which submitted liquidity submission
	// which still haven't been deployed even once.
	pendings *SnapshotablePendingProvisions

	// the maximum number of liquidity orders to be created on
	// each shape
	maxShapesSize int64

	// this is the max fee that can be specified
	maxFee num.Decimal

	// this is the ratio between 10^{asset_dp} / 10^{market_dp}
	priceFactor *num.Uint

	// lpPartyOrders are ones that have been removed from the book but we need to know
	// what they were to calculate new LP orders. They are cleared once the new LP orders are
	// ready to deploy
	lpPartyOrders map[string][]*types.Order

	// fields used for liquidity score calculation (quality of deployed orders)
	avgScores map[string]num.Decimal
	nAvg      int64 // counter for the liquidity score running average
}

// NewEngine returns a new Liquidity Engine.
func NewEngine(config Config,
	log *logging.Logger,
	timeService TimeService,
	broker Broker,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	orderBook OrderBook,
	asset string,
	marketID string,
	stateVarEngine StateVarEngine,
	priceFactor *num.Uint,
	positionFactor num.Decimal,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		marketID:    marketID,
		log:         log,
		timeService: timeService,
		broker:      broker,
		// tick size to be used by the supplied engine should actually be in asset decimal
		suppliedEngine: supplied.NewEngine(riskModel, priceMonitor, asset, marketID, stateVarEngine, log, positionFactor),
		orderBook:      orderBook,

		// parameters
		stakeToObligationFactor: num.DecimalFromInt64(1),
		maxShapesSize:           5, // set it to the same default than the netparams
		maxFee:                  num.DecimalFromInt64(1),
		priceFactor:             priceFactor,
		// provisions related state
		provisions: newSnapshotableProvisionsPerParty(),
		pendings:   newSnapshotablePendingProvisions(),

		// lp orders that have been removed from the book but yet to be redeployed
		lpPartyOrders: map[string][]*types.Order{},
	}
	e.ResetAverageLiquidityScores() // initialise

	return e
}

func (e *Engine) SetGetStaticPricesFunc(f func() (num.Decimal, num.Decimal, error)) {
	e.suppliedEngine.SetGetStaticPricesFunc(f)
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

func (e *Engine) GetPending() []string {
	pending := make([]string, 0, len(e.pendings.m))
	for v := range e.pendings.m {
		pending = append(pending, v)
	}
	sort.Strings(pending)
	return pending
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
) error {
	lp, ok := e.provisions.Get(party)
	if !ok {
		return errors.New("party have no liquidity provision orders")
	}

	var orderStatus types.OrderStatus
	switch status {
	case types.LiquidityProvisionStatusCancelled:
		orderStatus = types.OrderStatusCancelled
	case types.LiquidityProvisionStatusStopped:
		orderStatus = types.OrderStatusStopped
	case types.LiquidityProvisionStatusRejected:
		orderStatus = types.OrderStatusRejected
	default:
		e.log.Panic("unsupported liquidity provisions status", logging.String("status", status.String()))
	}

	now := e.timeService.GetTimeNow().UnixNano()

	// get list of orders in the book so we do not send duplicates events
	cancels := e.orderBook.GetLiquidityOrders(party)
	sort.Slice(cancels, func(i, j int) bool {
		return cancels[i].ID < cancels[j].ID
	})

	cancelsM := map[string]struct{}{}
	for _, c := range cancels {
		cancelsM[c.ID] = struct{}{}
	}

	evts := e.getCancelAllLiquidityOrders(ctx, lp, cancelsM, orderStatus, now)
	e.broker.SendBatch(evts)

	lp.Status = status
	lp.UpdatedAt = now
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	// now delete all stuff
	e.provisions.Delete(party)
	e.pendings.Delete(party)
	return nil
}

// IsLiquidityProvider returns true if the party hold any liquidity commitmement.
func (e *Engine) IsLiquidityProvider(party string) bool {
	_, ok := e.provisions.Get(party)
	return ok
}

// RejectLiquidityProvision removes a parties commitment of liquidity.
func (e *Engine) RejectLiquidityProvision(ctx context.Context, party string) error {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusRejected)
}

// CancelLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any.
func (e *Engine) CancelLiquidityProvision(ctx context.Context, party string) error {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusCancelled)
}

// StopLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any.
func (e *Engine) StopLiquidityProvision(ctx context.Context, party string) error {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusStopped)
}

// ProvisionsPerParty returns the registered a map of party-id -> LiquidityProvision.
func (e *Engine) ProvisionsPerParty() ProvisionsPerParty {
	return e.provisions.ProvisionsPerParty
}

// SaveLPOrders sets LP orders that have been cancelled from the book but we need to know
// what they were to recalculate and redeploy.
func (e *Engine) SaveLPOrders() {
	for _, p := range e.provisions.ProvisionsPerParty {
		e.lpPartyOrders[p.Party] = e.orderBook.GetLiquidityOrders(p.Party)
	}
}

func (e *Engine) ClearLPOrders() {
	e.lpPartyOrders = map[string][]*types.Order{}
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
		CreatedAt:        e.timeService.GetTimeNow().UnixNano(),
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
func (e *Engine) SubmitLiquidityProvision(
	ctx context.Context,
	lps *types.LiquidityProvisionSubmission,
	party string,
	idgen IDGen,
) error {
	if err := e.ValidateLiquidityProvisionSubmission(lps, false); err != nil {
		e.rejectLiquidityProvisionSubmission(ctx, lps, party, idgen.NextID())
		return err
	}

	if lp := e.LiquidityProvisionByPartyID(party); lp != nil {
		return ErrLiquidityProvisionAlreadyExists
	}

	var (
		now = e.timeService.GetTimeNow().UnixNano()
		lp  = &types.LiquidityProvision{
			ID:        idgen.NextID(),
			MarketID:  lps.MarketID,
			Party:     party,
			CreatedAt: now,
			Fee:       lps.Fee,
			Status:    types.LiquidityProvisionStatusRejected,
			Reference: lps.Reference,
			Version:   1,
		}
	)

	// regardless of the final operation (create,update or delete) we finish
	// sending an event.
	defer func() {
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	}()

	e.provisions.Set(party, lp)
	e.pendings.Add(party)
	lp.UpdatedAt = now
	lp.CommitmentAmount = lps.CommitmentAmount
	lp.Status = types.LiquidityProvisionStatusPending

	orderEvts := e.SetShapesReferencesOnLiquidityProvision(ctx, lp, lps.Buys, lps.Sells, idgen)

	// seed the dummy orders with the generated IDs in order to avoid broken references
	e.broker.SendBatch(orderEvts)

	return nil
}

func (e *Engine) SetShapesReferencesOnLiquidityProvision(
	ctx context.Context,
	lp *types.LiquidityProvision,
	buys []*types.LiquidityOrder,
	sells []*types.LiquidityOrder,
	idGen IDGen,
) []events.Event {
	// this order is just a stub to send to the id generator,
	// and get an ID assigned per references in the shapes
	lp.Buys = make([]*types.LiquidityOrderReference, 0, len(buys))
	orderEvts := make([]events.Event, 0, len(buys)+len(sells))

	for _, buy := range buys {
		order := &types.Order{
			ID:                   idGen.NextID(),
			MarketID:             e.marketID,
			Party:                lp.Party,
			Side:                 types.SideBuy,
			Price:                num.UintZero(),
			Status:               types.OrderStatusStopped,
			Reference:            lp.Reference,
			LiquidityProvisionID: lp.ID,
			CreatedAt:            lp.CreatedAt,
			Type:                 types.OrderTypeLimit,
		}
		lp.Buys = append(lp.Buys, &types.LiquidityOrderReference{
			OrderID:        order.ID,
			LiquidityOrder: buy,
		})
		orderEvts = append(orderEvts, events.NewOrderEvent(ctx, order))
	}

	lp.Sells = make([]*types.LiquidityOrderReference, 0, len(sells))
	for _, sell := range sells {
		order := &types.Order{
			ID:                   idGen.NextID(),
			MarketID:             e.marketID,
			Party:                lp.Party,
			Side:                 types.SideSell,
			Price:                num.UintZero(),
			Status:               types.OrderStatusStopped,
			Reference:            lp.Reference,
			LiquidityProvisionID: lp.ID,
			CreatedAt:            lp.CreatedAt,
			Type:                 types.OrderTypeLimit,
		}
		lp.Sells = append(lp.Sells, &types.LiquidityOrderReference{
			OrderID:        order.ID,
			LiquidityOrder: sell,
		})
		orderEvts = append(orderEvts, events.NewOrderEvent(ctx, order))
	}

	return orderEvts
}

// LiquidityProvisionByPartyID returns the LP associated to a Party if any.
// If not, it returns nil.
func (e *Engine) LiquidityProvisionByPartyID(partyID string) *types.LiquidityProvision {
	lp, _ := e.provisions.Get(partyID)
	return lp
}

// CreateInitialOrders returns two slices of orders, one for orders to be
// created and the other for orders to be updated.
func (e *Engine) CreateInitialOrders(
	ctx context.Context,
	minLpPrice, maxLpPrice *num.Uint,
	party string,
	repriceFn RepriceOrder,
) []*types.Order {
	// ignoring amends as there won't be any since we kill all the orders first
	creates, _ := e.createOrUpdateForParty(ctx,
		minLpPrice, maxLpPrice, party, repriceFn)
	return creates
}

// UndeployLPs is called when a reference price is no longer available. LP orders should all be parked/set to pending
// and should be redeployed once possible. Pass in updated orders and update internal records first...
func (e *Engine) UndeployLPs(ctx context.Context, orders []*types.Order) []*ToCancel {
	provisions := e.provisions.Slice()
	cancels := make([]*ToCancel, 0, len(provisions)*2) // one for each side
	evts := make([]events.Event, 0, len(provisions))
	for _, lp := range provisions {
		if lp.Status != types.LiquidityProvisionStatusActive {
			continue
		}
		buys := make([]*supplied.LiquidityOrder, 0, len(lp.Buys))
		sells := make([]*supplied.LiquidityOrder, 0, len(lp.Sells))
		for _, o := range lp.Buys {
			buys = append(buys, &supplied.LiquidityOrder{
				OrderID: o.OrderID,
				Details: o.LiquidityOrder,
			})
		}
		for _, o := range lp.Sells {
			sells = append(sells, &supplied.LiquidityOrder{
				OrderID: o.OrderID,
				Details: o.LiquidityOrder,
			})
		}
		if cb := e.undeployOrdersFromShape(lp.Party, buys, types.SideBuy); cb != nil {
			cancels = append(cancels, cb)
		}
		if cs := e.undeployOrdersFromShape(lp.Party, sells, types.SideSell); cs != nil {
			cancels = append(cancels, cs)
		}
		// set as undeployed so we can redeploy it once the pegs become available again
		lp.Status = types.LiquidityProvisionStatusUndeployed
		evts = append(evts, events.NewLiquidityProvisionEvent(ctx, lp))
	}

	e.broker.SendBatch(evts)
	return cancels
}

// Update gets the order changes.
// It keeps track of all LP orders.
func (e *Engine) Update(
	ctx context.Context,
	minLpPrice, maxLpPrice *num.Uint,
	repriceFn RepriceOrder,
) ([]*types.Order, []*ToCancel) {
	var (
		newOrders []*types.Order
		toCancel  []*ToCancel
	)
	for _, lp := range e.provisions.Slice() {
		creates, cancels := e.createOrUpdateForParty(ctx, minLpPrice, maxLpPrice, lp.Party, repriceFn)
		newOrders = append(newOrders, creates...)
		if !cancels.Empty() {
			toCancel = append(toCancel, cancels)
		}
	}
	return newOrders, toCancel
}

// CalculateSuppliedStake returns the sum of commitment amounts from all the liquidity providers.
func (e *Engine) CalculateSuppliedStake() *num.Uint {
	ss := num.UintZero()
	for _, v := range e.provisions.ProvisionsPerParty {
		ss.AddSum(v.CommitmentAmount)
	}
	return ss
}

func (e *Engine) createOrUpdateForParty(
	ctx context.Context,
	minLpPrice, maxLpPrice *num.Uint,
	party string,
	repriceFn RepriceOrder,
) (ordres []*types.Order, _ *ToCancel) {
	lp := e.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return nil, nil
	}

	var (
		obligation, _  = num.UintFromDecimal(lp.CommitmentAmount.ToDecimal().Mul(e.stakeToObligationFactor).Round(0))
		buysShape      = make([]*supplied.LiquidityOrder, 0, len(lp.Buys))
		sellsShape     = make([]*supplied.LiquidityOrder, 0, len(lp.Sells))
		repriceFailure bool
		lpChanged      bool
	)

	for _, buy := range lp.Buys {
		order := &supplied.LiquidityOrder{
			OrderID: buy.OrderID,
			Details: buy.LiquidityOrder,
		}
		if price, err := repriceFn(types.SideBuy, buy.LiquidityOrder.Reference, buy.LiquidityOrder.Offset.Clone()); err != nil {
			e.log.Debug("Building Buy Shape", logging.Error(err))
			repriceFailure = true
		} else {
			order.Price = price
		}
		buysShape = append(buysShape, order)
	}

	for _, sell := range lp.Sells {
		order := &supplied.LiquidityOrder{
			OrderID: sell.OrderID,
			Details: sell.LiquidityOrder,
		}
		if price, err := repriceFn(types.SideSell, sell.LiquidityOrder.Reference, sell.LiquidityOrder.Offset.Clone()); err != nil {
			e.log.Debug("Building Sell Shape", logging.Error(err))
			repriceFailure = true
		} else {
			order.Price = price
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
			lpChanged = true
		}
	} else {
		// Create a slice shaped copy of the orders
		partyOrders := e.orderBook.GetOrdersPerParty(party)
		orders := make([]*types.Order, 0, len(partyOrders))
		for _, order := range partyOrders {
			if !order.IsLiquidityOrder() && order.Status == vega.Order_STATUS_ACTIVE {
				orders = append(orders, order)
			}
		}

		e.suppliedEngine.CalculateLiquidityImpliedVolumes(
			obligation,
			orders,
			minLpPrice, maxLpPrice,
			buysShape, sellsShape,
		)

		needsCreateBuys, needsUpdateBuys = e.createOrdersFromShape(
			party, buysShape, types.SideBuy)
		needsCreateSells, needsUpdateSells = e.createOrdersFromShape(
			party, sellsShape, types.SideSell)

		if lp.Status != types.LiquidityProvisionStatusActive {
			lp.Status = types.LiquidityProvisionStatusActive
			lpChanged = true
		}
	}

	// fields in the lp might have changed so we re-set it to trigger the snapshot `changed` flag
	e.provisions.Set(party, lp)
	if lpChanged {
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	}
	return append(needsCreateBuys, needsCreateSells...),
		needsUpdateBuys.Merge(needsUpdateSells)
}

func (e *Engine) buildOrder(side types.Side, price *num.Uint, partyID, marketID string, size uint64, ref string, lpID string) *types.Order {
	op := price.Clone()
	op.Div(op, e.priceFactor)
	order := &types.Order{
		MarketID:             marketID,
		Side:                 side,
		Price:                price.Clone(),
		OriginalPrice:        op,
		Party:                partyID,
		Size:                 size,
		Remaining:            size,
		Type:                 types.OrderTypeLimit,
		TimeInForce:          types.OrderTimeInForceGTC,
		Reference:            ref,
		LiquidityProvisionID: lpID,
	}
	p, _ := e.provisions.Get(partyID)
	return order.Create(p.CreatedAt)
}

func (e *Engine) undeployOrdersFromShape(
	party string, shape []*supplied.LiquidityOrder, side types.Side,
) *ToCancel {
	lm := map[string]*types.Order{}
	for _, lo := range e.orderBook.GetLiquidityOrders(party) {
		lm[lo.ID] = lo
	}
	for _, lo := range e.lpPartyOrders[party] {
		lm[lo.ID] = lo
	}

	var (
		toCancel = &ToCancel{
			Party: party,
		}
		lp = e.LiquidityProvisionByPartyID(party)
	)

	for i, o := range shape {
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
			delete(lm, ref.OrderID)
		}
	}

	return toCancel
}

func (e *Engine) createOrdersFromShape(
	party string, supplied []*supplied.LiquidityOrder, side types.Side,
) ([]*types.Order, *ToCancel) {
	lm := map[string]*types.Order{}
	for _, lo := range e.orderBook.GetLiquidityOrders(party) {
		lm[lo.ID] = lo
	}
	for _, lo := range e.lpPartyOrders[party] {
		lm[lo.ID] = lo
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
		}

		// We either don't need this order anymore or
		// we have just nothing to do about it.
		if o.LiquidityImpliedVolume == 0 ||
			// we check if the order was not nil, which mean we already had a deployed order
			// if the order as not traded, and the size haven't changed, then we have nothing
			// to do about it. If the size has changed, then we will want to recreate one.
			(order != nil && (!order.HasTraded() && order.Size == o.LiquidityImpliedVolume && order.Price.EQ(o.Price))) ||
			// we check o.Price == 0 just to make sure we are able to price
			// the order, in which case the size will have been calculated
			// properly by the engine.
			o.Price.IsZero() {
			continue
		}

		// At this point the order will either already exists
		// or not, and we'll want to re-create
		// then we create the new order
		order = e.buildOrder(side, o.Price, party, e.marketID, o.LiquidityImpliedVolume, lp.Reference, lp.ID)
		order.ID = ref.OrderID
		newOrders = append(newOrders, order)
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
			case types.PeggedReferenceMid:
				if lo.Offset.IsZero() {
					return errors.New("order in buy side shape offset must be > 0")
				}
			}
		} else {
			switch lo.Reference {
			case types.PeggedReferenceBestAsk:
			case types.PeggedReferenceBestBid:
				return errors.New("order in buy side shape with best ask price reference")
			case types.PeggedReferenceMid:
				if lo.Offset.IsZero() {
					return errors.New("order in sell shape offset must be > 0")
				}
			}
		}
	}
	return nil
}

func (e *Engine) GetAverageLiquidityScores() map[string]num.Decimal {
	return e.avgScores
}

func (e *Engine) UpdateAverageLiquidityScores(bestBid, bestAsk num.Decimal, minLpPrice, maxLpPrice *num.Uint) {
	current, total := e.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	nLps := len(current)
	if nLps == 0 {
		return
	}

	// normalise first
	equalFraction := num.DecimalOne().Div(num.DecimalFromInt64(int64(nLps)))
	for k, v := range current {
		if total.IsZero() {
			current[k] = equalFraction
		} else {
			current[k] = v.Div(total)
		}
	}

	if e.nAvg > 1 {
		n := num.DecimalFromInt64(e.nAvg)
		nMinusOneOverN := n.Sub(num.DecimalOne()).Div(n)

		for k, vNew := range current {
			// if not found then it defaults to 0
			vOld := e.avgScores[k]
			current[k] = vOld.Mul(nMinusOneOverN).Add(vNew.Div(n))
		}
	}

	for k := range current {
		current[k] = current[k].Round(10)
	}

	// always overwrite with latest to automatically remove LPs that are no longer ACTIVE from the list
	e.avgScores = current
	e.nAvg++
}

func (e *Engine) ResetAverageLiquidityScores() {
	e.avgScores = make(map[string]num.Decimal, len(e.avgScores))
	e.nAvg = 1
}

// GetCurrentLiquidityScores returns volume-weighted probability of trading per each LP's deployed orders.
func (e *Engine) GetCurrentLiquidityScores(bestBid, bestAsk num.Decimal, minLpPrice, maxLpPrice *num.Uint) (map[string]num.Decimal, num.Decimal) {
	provs := e.provisions.Slice()
	t := num.DecimalZero()
	r := make(map[string]num.Decimal, len(provs))
	for _, p := range provs {
		if p.Status != vega.LiquidityProvision_STATUS_ACTIVE {
			continue
		}
		orders := e.getAllActiveOrders(p.Party)
		l := e.suppliedEngine.CalculateLiquidityScore(orders, bestBid, bestAsk, minLpPrice, maxLpPrice)
		r[p.Party] = l
		t = t.Add(l)
	}
	return r, t
}

func (e *Engine) getAllActiveOrders(party string) []*types.Order {
	partyOrders := e.orderBook.GetOrdersPerParty(party)
	orders := make([]*types.Order, 0, len(partyOrders))
	for _, order := range partyOrders {
		if order.Status == vega.Order_STATUS_ACTIVE {
			orders = append(orders, order)
		}
	}
	return orders
}

func (e *Engine) IsPoTInitialised() bool {
	return e.suppliedEngine.IsPoTInitialised()
}

func (e *Engine) UpdateMarketConfig(model risk.Model, monitor PriceMonitor) {
	e.suppliedEngine.UpdateMarketConfig(model, monitor)
}

// GetLPShapeCount returns the total number of LP shapes.
func (e *Engine) GetLPShapeCount() uint64 {
	var total uint64
	for _, v := range e.provisions.Slice() {
		total += uint64(len(v.Buys) + len(v.Sells))
	}
	return total
}

func (e *Engine) getCancelAllLiquidityOrders(
	ctx context.Context,
	lp *types.LiquidityProvision,
	excludeIDs map[string]struct{},
	cancelWithStatus types.OrderStatus,
	canceledAt int64,
) []events.Event {
	if excludeIDs == nil {
		excludeIDs = map[string]struct{}{}
	}

	// here we will cancel the orders which are not in the book
	evts := []events.Event{}
	for _, o := range lp.Buys {
		if _, ok := excludeIDs[o.OrderID]; ok {
			// this order was on the book
			// nothing to do, it'll be cancelled
			// later by the market hopefully
			continue
		}

		evts = append(evts, events.NewOrderEvent(ctx, &types.Order{
			ID:                   o.OrderID,
			MarketID:             e.marketID,
			Party:                lp.Party,
			Side:                 types.SideBuy,
			Price:                num.UintZero(),
			Size:                 0,
			Status:               cancelWithStatus,
			Reference:            lp.Reference,
			LiquidityProvisionID: lp.ID,
			CreatedAt:            lp.CreatedAt,
			UpdatedAt:            canceledAt,
			Type:                 types.OrderTypeLimit,
			TimeInForce:          types.OrderTimeInForceGTC,
		}))
	}

	for _, o := range lp.Sells {
		if _, ok := excludeIDs[o.OrderID]; ok {
			// this order was on the book
			// nothing to do, it'll be cancelled
			// later by the market hopefully
			continue
		}
		evts = append(evts, events.NewOrderEvent(ctx, &types.Order{
			ID:                   o.OrderID,
			MarketID:             e.marketID,
			Party:                lp.Party,
			Side:                 types.SideSell,
			Price:                num.UintZero(),
			Size:                 0,
			Status:               cancelWithStatus,
			Reference:            lp.Reference,
			LiquidityProvisionID: lp.ID,
			CreatedAt:            lp.CreatedAt,
			UpdatedAt:            canceledAt,
			Type:                 types.OrderTypeLimit,
			TimeInForce:          types.OrderTimeInForceGTC,
		}))
	}

	return evts
}
