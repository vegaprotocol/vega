package liquidity

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrLiquidityProvisionDoesNotExist = errors.New("liquidity provision does not exist")
	ErrEmptyShape                     = errors.New("liquidity provision contains an empty shape")
)

//go:generate mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/liquidity Broker,RiskModel,PriceMonitor,IDGen

// Broker - event bus
type Broker interface {
	Send(e events.Event)
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
type RepricePeggedOrder func(order *types.PeggedOrder) (uint64, error)

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

	// orders stores all the market orders (except the liquidity orders) explicitly submited by a given party.
	// indexed as: map of PartyID -> OrderID -> order to easy access
	orders map[string]map[string]*types.Order

	// liquidityOrder stores the orders generated to satisfy the liquidity commitment of a given party.
	// indexed as: map of PartyID -> OrdersID -> order
	liquidityOrders map[string]map[string]*types.Order
}

// NewEngine returns a new Liquidity Engine.
func NewEngine(
	log *logging.Logger,
	broker Broker,
	idGen IDGen,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	marketID string,
) *Engine {
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
	}
}

// OnChainTimeUpdate updates the internal engine current time
func (e *Engine) OnChainTimeUpdate(ctx context.Context, now time.Time) {
	e.currentTime = now
}

func (e *Engine) OnSuppliedStakeToObligationFactorUpdate(v float64) {
	e.stakeToObligationFactor = v
}

func (e *Engine) CancelLiquidityProvision(ctx context.Context, party string) error {
	lp := e.provisions[party]
	if lp == nil {
		return errors.New("party have no liquidity provision orders")
	}

	lp.Status = types.LiquidityProvision_LIQUIDITY_PROVISION_STATUS_REJECTED
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	// now delete all stuff
	delete(e.liquidityOrders, party)
	delete(e.orders, party)
	return nil
}

// ProvisionsPerParty returns the resgistered a map of party-id -> LiquidityProvision.
func (e *Engine) ProvisionsPerParty() ProvisionsPerParty {
	return e.provisions
}

// SubmitLiquidityProvision handles a new liquidity provision submission.
// It's used to create, update or delete a LiquidityProvision.
// The LiquidityProvision is created if submited for the first time, updated if a
// previous one was created for the same PartyId or deleted (if exists) when
// the CommitmentAmount is set to 0.
func (e *Engine) SubmitLiquidityProvision(ctx context.Context, lps *types.LiquidityProvisionSubmission, party, id string) error {
	var (
		lp  *types.LiquidityProvision = e.LiquidityProvisionByPartyID(party)
		now                           = e.currentTime.UnixNano()
	)

	if len(lps.Buys) == 0 && len(lps.Sells) == 0 {
		return ErrEmptyShape
	}

	// regardless of the final operaion (create,update or delete) we finish
	// sending an event.
	defer func() {
		evt := events.NewLiquidityProvisionEvent(ctx, lp)
		e.broker.Send(evt)
	}()

	// We are trying to delete the provision
	if lps.CommitmentAmount == 0 {
		// Reject a delete attempt for a non existing LP.
		if lp == nil {
			lp = &types.LiquidityProvision{
				Id:        id,
				MarketID:  lps.MarketID,
				PartyID:   party,
				CreatedAt: now,
				Status:    types.LiquidityProvision_LIQUIDITY_PROVISION_STATUS_REJECTED,
			}
			return ErrLiquidityProvisionDoesNotExist
		}
		// Cancel the request
		lp.Status = types.LiquidityProvision_LIQUIDITY_PROVISION_STATUS_CANCELLED
		lp.CommitmentAmount = 0
		delete(e.provisions, party)
		return nil
	}

	if lp == nil {
		lp = &types.LiquidityProvision{
			Id:        id,
			MarketID:  lps.MarketID,
			PartyID:   party,
			CreatedAt: now,
		}

		e.provisions[party] = lp
		e.orders[party] = map[string]*types.Order{}
		e.liquidityOrders[party] = map[string]*types.Order{}
	}

	lp.UpdatedAt = now
	lp.CommitmentAmount = lps.CommitmentAmount
	lp.Status = types.LiquidityProvision_LIQUIDITY_PROVISION_STATUS_ACTIVE

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

	return nil
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

// CreateInitialOrders returns two slices of orders, one are the one to be
// created and the other the one to be updates.
func (e *Engine) CreateInitialOrders(markPrice uint64, party string, orders []*types.Order, repriceFn RepricePeggedOrder) ([]*types.Order, []*types.OrderAmendment, error) {
	// update our internal orders
	e.updatePartyOrders(party, orders)
	creates, amendments, err := e.createOrUpdateForParty(markPrice, party, repriceFn)
	return creates, amendments, err
}

// Update gets the order changes.
// It keeps track of all LP orders.
func (e *Engine) Update(markPrice uint64, repriceFn RepricePeggedOrder, orders []*types.Order) ([]*types.Order, []*types.OrderAmendment, error) {
	var (
		newOrders  []*types.Order
		amendments []*types.OrderAmendment
	)

	for party, orders := range Orders(orders).ByParty() {
		// update our internal orders
		e.updatePartyOrders(party, orders)

		creates, updates, err := e.createOrUpdateForParty(markPrice, party, repriceFn)
		if err != nil {
			return nil, nil, err
		}

		newOrders = append(newOrders, creates...)
		amendments = append(amendments, updates...)
	}

	return newOrders, amendments, nil
}

// CalculateSuppliedStake returns the sum of commitment amounts from all the liquidity providers
func (e *Engine) CalculateSuppliedStake() uint64 {
	var ss uint64 = 0
	for _, v := range e.provisions {
		ss += v.CommitmentAmount
	}
	return ss
}

func (e *Engine) createOrUpdateForParty(markPrice uint64, party string, repriceFn RepricePeggedOrder) ([]*types.Order, []*types.OrderAmendment, error) {
	lp := e.LiquidityProvisionByPartyID(party)
	if lp == nil {
		return nil, nil, nil
	}

	// Create a slice shaped copy of the orders
	buyOrders := make([]*types.Order, 0, len(e.orders[party])/2)
	sellOrders := make([]*types.Order, 0, len(e.orders[party])/2)
	for _, order := range e.orders[party] {
		if order.Side == types.Side_SIDE_BUY {
			buyOrders = append(buyOrders, order)
		} else {
			sellOrders = append(sellOrders, order)
		}
	}

	obligation := float64(lp.CommitmentAmount) * e.stakeToObligationFactor
	var (
		buysShape  = make([]*supplied.LiquidityOrder, 0, len(lp.Buys))
		sellsShape = make([]*supplied.LiquidityOrder, 0, len(lp.Sells))
	)

	for _, buy := range lp.Buys {
		pegged := &types.PeggedOrder{
			Reference: buy.LiquidityOrder.Reference,
			Offset:    buy.LiquidityOrder.Offset,
		}
		price, err := repriceFn(pegged)
		if err != nil {
			continue
		}
		buysShape = append(buysShape, &supplied.LiquidityOrder{
			OrderID:    buy.OrderID,
			Price:      price,
			Proportion: uint64(buy.LiquidityOrder.Proportion),
		})
	}

	for _, sell := range lp.Sells {
		pegged := &types.PeggedOrder{
			Reference: sell.LiquidityOrder.Reference,
			Offset:    sell.LiquidityOrder.Offset,
		}
		price, err := repriceFn(pegged)
		if err != nil {
			continue
		}
		sellsShape = append(sellsShape, &supplied.LiquidityOrder{
			OrderID:    sell.OrderID,
			Price:      price,
			Proportion: uint64(sell.LiquidityOrder.Proportion),
		})
	}

	if err := e.suppliedEngine.CalculateLiquidityImpliedVolumes(
		float64(markPrice),
		obligation,
		buyOrders, sellOrders,
		buysShape, sellsShape,
	); err != nil {
		return nil, nil, err
	}

	needsCreateBuys, needsUpdateBuys := e.createOrdersFromShape(party, buysShape, types.Side_SIDE_BUY)
	needsCreateSells, needsUpdateSells := e.createOrdersFromShape(party, sellsShape, types.Side_SIDE_SELL)

	return append(needsCreateBuys, needsCreateSells...),
		append(needsUpdateBuys, needsUpdateSells...),
		nil
}

func buildOrder(side types.Side, pegged *types.PeggedOrder, price uint64, partyID, marketID string, size uint64) *types.Order {
	return &types.Order{
		MarketID:    marketID,
		Side:        side,
		PeggedOrder: pegged,
		Price:       price,
		PartyID:     partyID,
		Size:        size,
		Remaining:   size,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
	}
}

func (e *Engine) createOrdersFromShape(party string, supplied []*supplied.LiquidityOrder, side types.Side) ([]*types.Order, []*types.OrderAmendment) {
	lm := e.liquidityOrders[party]
	lp := e.LiquidityProvisionByPartyID(party)

	var (
		newOrders  []*types.Order
		amendments []*types.OrderAmendment
	)

	for i, o := range supplied {
		order := lm[o.OrderID]
		var ref *types.LiquidityOrderReference
		if side == types.Side_SIDE_BUY {
			ref = lp.Buys[i]
		} else {
			ref = lp.Sells[i]
		}

		if order == nil {
			if o.LiquidityImpliedVolume == 0 {
				continue
			}

			p := &types.PeggedOrder{
				Reference: ref.LiquidityOrder.Reference,
				Offset:    ref.LiquidityOrder.Offset,
			}
			order = buildOrder(side, p, o.Price, party, e.marketID, o.LiquidityImpliedVolume)
			e.idGen.SetID(order)
			newOrders = append(newOrders, order)
			lm[order.Id] = order
			ref.OrderID = order.Id
			continue
		}

		if o.LiquidityImpliedVolume == 0 {
			delete(lm, ref.OrderID)
			ref.OrderID = ""
		}

		if newSize := o.LiquidityImpliedVolume; newSize != order.Size {
			amendments = append(amendments, order.AmendSize(int64(newSize)))
		}
	}

	return newOrders, amendments
}
