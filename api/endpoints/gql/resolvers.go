package gql

import (
	"context"
	"vega/api"
	"vega/msg"
	"errors"
)

type resolverRoot struct {
	orderService api.OrderService
	tradeService api.TradeService
}

func NewResolverRoot(orderService api.OrderService, tradeService api.TradeService) *resolverRoot {
	return &resolverRoot{
		orderService: orderService,
		tradeService: tradeService,
	}
}

func (r *resolverRoot) Query() QueryResolver {
	return (*MyQueryResolver)(r)
}
func (r *resolverRoot) Order() OrderResolver {
	return (*MyOrderResolver)(r)
}
func (r *resolverRoot) Trade() TradeResolver {
	return (*MyTradeResolver)(r)
}
func (r *resolverRoot) Vega() VegaResolver {
	return (*MyVegaResolver)(r)
}

// BEGIN: Query Resolver

type MyQueryResolver resolverRoot

func (r *MyQueryResolver) Vega(ctx context.Context) (Vega, error) {
	var vega = Vega{}
	return vega, nil
}

// END: Query Resolver


type MyVegaResolver resolverRoot

func (r *MyVegaResolver) Markets(ctx context.Context, obj *Vega, name *string) ([]Market, error) {
	if name == nil {
		return nil, errors.New("All markets for VEGA platform not implemented")
	}

	// Look for orders for market (will validate market internally)
	orders, err := r.orderService.GetByMarket(ctx, *name, 1000)
	if err != nil {
		return nil, err
	}
	
	valOrders := make([]msg.Order, 0)
	for _, v := range orders {
		valOrders = append(valOrders, *v)
	}
	
	var markets = make([]Market, 0)
	var market = Market{
		Name: *name,
		Orders: valOrders,
	}
	markets = append(markets, market)
	
	return markets, nil
}

func (r *MyVegaResolver) Parties(ctx context.Context, obj *Vega, name *string) ([]Party, error) {
	if name == nil {
		return nil, errors.New("All parties for VEGA platform not implemented")
	}

	// Look for orders for party (will validate market internally)
	orders, err := r.orderService.GetByParty(ctx, *name, 1000)
	if err != nil {
		return nil, err
	}

	valOrders := make([]msg.Order, 0)
	for _, v := range orders {
		valOrders = append(valOrders, *v)
	}
	var parties = make([]Party, 0)
	var party = Party{
		Name: *name,
		Orders: valOrders,
	}
	parties = append(parties, party)
	
	return parties, nil
}


// BEGIN: Order Resolver

type MyOrderResolver resolverRoot

func (r *MyOrderResolver) Price(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Price), nil
}
func (r *MyOrderResolver) Type(ctx context.Context, obj *msg.Order) (OrderType, error) {
	return OrderType(obj.Type.String()), nil
}
func (r *MyOrderResolver) Side(ctx context.Context, obj *msg.Order) (Side, error) {
	return Side(obj.Side.String()), nil
}
func (r *MyOrderResolver) Market(ctx context.Context, obj *msg.Order) (Market, error) {
	return Market {
		Name: obj.Market,
	}, nil
}
func (r *MyOrderResolver) Size(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Size), nil
}
func (r *MyOrderResolver) Remaining(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Remaining), nil
}
func (r *MyOrderResolver) Timestamp(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Timestamp), nil
}
func (r *MyOrderResolver) Status(ctx context.Context, obj *msg.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
}

// END: Order Resolver


// BEGIN: Trade Resolver

type MyTradeResolver resolverRoot

func (r *MyTradeResolver) Market(ctx context.Context, obj *msg.Trade) (Market, error) {
	return Market{Name: obj.Market}, nil
}
func (r *MyTradeResolver) Aggressor(ctx context.Context, obj *msg.Trade) (Side, error) {
	return Side(obj.Aggressor.String()), nil
}
func (r *MyTradeResolver) Price(ctx context.Context, obj *msg.Trade) (int, error) {
	return int(obj.Price), nil
}
func (r *MyTradeResolver) Size(ctx context.Context, obj *msg.Trade) (int, error) {
	return int(obj.Size), nil
}
func (r *MyTradeResolver) Timestamp(ctx context.Context, obj *msg.Trade) (int, error) {
	return int(obj.Timestamp), nil
}

// END: Trade Resolver
