package matching

import "vega/proto"

type Config struct {
	Quiet                  bool
	TradeChans             []chan msg.Trade
	OrderChans             []chan msg.Order
	OrderConfirmationChans []chan msg.OrderConfirmation
}

func DefaultConfig() Config {
	return Config{
		Quiet:                  false,
		OrderChans:             []chan msg.Order{},
		TradeChans:             []chan msg.Trade{},
		OrderConfirmationChans: []chan msg.OrderConfirmation{},
	}
}
