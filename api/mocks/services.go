package mocks

type MockOrderService struct {
	ResultSuccess bool
	ResultError error
}

func (p *MockOrderService) CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error) {
	return p.ResultSuccess, p.ResultError
}
