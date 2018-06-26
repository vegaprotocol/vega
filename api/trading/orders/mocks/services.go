package mocks

import (
	"vega/api/trading/orders/models"
	"vega/datastore"
)

type MockOrderService struct {
	ResultSuccess bool
	ResultError error
}

func (p *MockOrderService) CreateOrder(order models.Order) (success bool, err error) {
	return p.ResultSuccess, p.ResultError
}

func (p *MockOrderService) Init(orderStore datastore.OrderStore) {

}

func (p *MockOrderService) GetOrders(market string) (orders []models.Order, err error) {
	 return nil, nil
}