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
	log            *logging.Logger
	broker         Broker
	idGen          IDGen
	suppliedEngine *supplied.Engine

	currentTime    time.Time
	suppliedFactor float64

	// state
	provisions map[string]*types.LiquidityProvision

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
) *Engine {
	return &Engine{
		log:             log,
		broker:          broker,
		idGen:           idGen,
		suppliedEngine:  supplied.NewEngine(riskModel, priceMonitor),
		suppliedFactor:  1,
		provisions:      map[string]*types.LiquidityProvision{},
		orders:          map[string]map[string]*types.Order{},
		liquidityOrders: map[string]map[string]*types.Order{},
	}
}

// OnChainTimeUpdate updates the internal engine current time
func (e *Engine) OnChainTimeUpdate(ctx context.Context, now time.Time) {
	e.currentTime = now
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

	if len(lps.Buys) == 0 || len(lps.Sells) == 0 {
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

// Orders provides convenience functions to a slice of *veaga/proto.Orders.
type Orders []*types.Order

// ByParty returns the orders grouped by it's PartyID
func (ords Orders) ByParty() map[string][]*types.Order {
	parties := map[string][]*types.Order{}
	for _, order := range ords {
		parties[order.PartyID] = append(parties[order.PartyID], order)
	}
	return parties
}

func (e *Engine) updatePartyOrders(partyID string, orders []*types.Order) {
	// These maps are created by SubmitLiquidityProvision
	m := e.orders[partyID]
	lm := e.liquidityOrders[partyID]

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

// Update gets the order changes.
// It keeps track of all LP orders.
func (e *Engine) Update(markPrice uint64, repriceFn RepricePeggedOrder, orders []*types.Order) ([]*types.Order, []*types.Order, error) {
	var (
		needsCreate []*types.Order
		needsUpdate []*types.Order
	)

	for party, orders := range Orders(orders).ByParty() {
		lp := e.LiquidityProvisionByPartyID(party)
		if lp == nil {
			continue
		}

		// update our internal orders
		e.updatePartyOrders(party, orders)

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

		obligation := float64(lp.CommitmentAmount) * e.suppliedFactor
		var (
			buysShape  = make([]*supplied.LiquidityOrder, len(lp.Buys))
			sellsShape = make([]*supplied.LiquidityOrder, len(lp.Sells))
		)

		for i, buy := range lp.Buys {
			pegged := &types.PeggedOrder{
				Reference: buy.LiquidityOrder.Reference,
				Offset:    buy.LiquidityOrder.Offset,
			}
			price, err := repriceFn(pegged)
			if err != nil {
				continue
			}
			buysShape[i] = &supplied.LiquidityOrder{
				OrderID:    buy.OrderID,
				Price:      price,
				Proportion: uint64(buy.LiquidityOrder.Proportion),
			}
		}

		for i, sell := range lp.Sells {
			pegged := &types.PeggedOrder{
				Reference: sell.LiquidityOrder.Reference,
				Offset:    sell.LiquidityOrder.Offset,
			}
			price, err := repriceFn(pegged)
			if err != nil {
				continue
			}
			sellsShape[i] = &supplied.LiquidityOrder{
				OrderID:    sell.OrderID,
				Price:      price,
				Proportion: uint64(sell.LiquidityOrder.Proportion),
			}
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
		needsCreate = append(needsCreate, needsCreateBuys...)
		needsCreate = append(needsCreate, needsCreateSells...)
		needsUpdate = append(needsUpdate, needsUpdateBuys...)
		needsUpdate = append(needsUpdate, needsUpdateSells...)
	}

	return needsCreate, needsUpdate, nil
}

func buildOrder(side types.Side, pegged *types.PeggedOrder, price uint64, partyID string, size uint64) *types.Order {
	return &types.Order{
		Side:        side,
		PeggedOrder: pegged,
		Price:       price,
		PartyID:     partyID,
		Size:        size,
		Remaining:   size,
	}
}

func (e *Engine) createOrdersFromShape(party string, supplied []*supplied.LiquidityOrder, side types.Side) ([]*types.Order, []*types.Order) {
	lm := e.liquidityOrders[party]
	lp := e.LiquidityProvisionByPartyID(party)

	var (
		needsCreate []*types.Order
		needsUpdate []*types.Order
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
			order = buildOrder(side, p, o.Price, party, o.LiquidityImpliedVolume)
			e.idGen.SetID(order)
			needsCreate = append(needsCreate, order)
			lm[order.Id] = order
			ref.OrderID = order.Id
			continue
		}

		if o.LiquidityImpliedVolume == 0 {
			delete(lm, ref.OrderID)
			ref.OrderID = ""
		}

		if o.LiquidityImpliedVolume != order.Size {
			order.Size = o.LiquidityImpliedVolume
			order.Remaining = o.LiquidityImpliedVolume
			needsUpdate = append(needsUpdate, order)
		}
	}

	return needsCreate, needsUpdate
}
