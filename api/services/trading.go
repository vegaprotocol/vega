package services

type TradingService interface {
	CreateOrder(string) string
}

type RpcTradingService struct {
}

func NewRpcTradingService() RpcTradingService {
	return RpcTradingService{}
}

func (p *RpcTradingService) CreateOrder(s string) string {
	return "RPC::CREATE::ORDER::SUCCESS"
}
