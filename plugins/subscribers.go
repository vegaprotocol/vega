package plugins

import (
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

// TradeSub subscription to the trade buffer
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_sub_mock.go -package mocks code.vegaprotocol.io/vega/plugins TradeSub
type TradeSub interface {
	Recv() <-chan []types.Trade
	Done() <-chan struct{}
}

// MarketSub subscription for the candles plugin to be aware of (new) markets
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_sub_mock.go -package mocks code.vegaprotocol.io/vega/plugins MarketSub
type MarketSub interface {
	Recv() <-chan []types.Market
	Done() <-chan struct{} // we're not using this ATM
}

// PosBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/pos_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PosBuffer
type PosBuffer interface {
	Subscribe() (<-chan []events.SettlePosition, int)
	Unsubscribe(int)
}
