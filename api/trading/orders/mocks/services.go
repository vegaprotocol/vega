package mocks

import (
	"vega/api/trading/orders/models"
	"vega/datastore"
)

type MockOrderService struct {
	ResultSuccess bool
	ResultError error
	ResultOrders []models.Order
	MockOrderStore datastore.OrderStore
}

func (p *MockOrderService) CreateOrder(order models.Order) (success bool, err error) {
	return p.ResultSuccess, p.ResultError
}

func (p *MockOrderService) Init(orderStore datastore.OrderStore) {
	p.MockOrderStore = orderStore
}

func (p *MockOrderService) GetOrders(market string) (orders []models.Order, err error) {
	 return p.ResultOrders, p.ResultError
}