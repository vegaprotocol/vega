package graphql

import (
	"vega/msg"
	"context"
	"vega/api"
)

type resolvers struct {
	orders []msg.Order
	orderService api.OrderService
}

type QueryResolver resolvers

func (r *resolvers) OrderQuery() OrderQueryResolver {
	return (*QueryResolver)(r)
}

func (r *resolvers) Order() OrderResolver {
	return (*QueryResolver)(r)
}


func (r *QueryResolver) Orders(ctx context.Context) ([]msg.Order, error) {
	orders, err := r.orderService.GetOrders(ctx,"BTC/DEC18", "", 99999)
	return orders, err
}

func (r *QueryResolver) Price(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Price), nil
}
func (r *QueryResolver) Type(ctx context.Context, obj *msg.Order) (OrderType, error) {
	return "GTT", nil
}
func (r *QueryResolver) Side(ctx context.Context, obj *msg.Order) (Side, error) {
	return "Buy", nil
}
func (r *QueryResolver) Market(ctx context.Context, obj *msg.Order) (Market, error) {
	return Market { obj.Market }, nil
}
func (r *QueryResolver) Size(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Size), nil
}
func (r *QueryResolver) Remaining(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Remaining), nil
}
func (r *QueryResolver) Timestamp(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Timestamp), nil
}

func NewQueryResolver(orderService api.OrderService) *resolvers {
	return &resolvers{
		orderService: orderService,
		orders: []msg.Order{},
	}
}