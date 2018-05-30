package mocks

import "vega/api/trading/orders/models"

type MockOrderService struct {
	ResultSuccess bool
	ResultError error
}

func (p *MockOrderService) CreateOrder(order models.Order) (success bool, err error) {
	return p.ResultSuccess, p.ResultError
}
