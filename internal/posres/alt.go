package posres

import (
	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/matching"
)

type Foo struct {
	marketID  string
	traders   []events.MarketPosition
	orderBook *matching.OrderBook
}

func New2(mkt string, traders []events.MarketPosition, orderBook *matching.Engine) *Foo {
	return &Foo{
		marketID:  mkt,
		traders:   traders,
		orderBook: orderBook,
	}
}
