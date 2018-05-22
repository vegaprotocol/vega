package mocks

import "vega/api/trading/orders"

type MockOrderService struct {
	ResultSuccess bool
	ResultError error
}

func (p *MockOrderService) CreateOrder(order orders.Order) (success bool, err error) {
	return p.ResultSuccess, p.ResultError
}
