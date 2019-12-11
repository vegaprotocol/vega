package buffer

import "context"

type Buffers struct {
	Trades  *TradeCh
	Orders  *OrderCh
	Markets *MarketCh
}

func New(ctx context.Context) *Buffers {
	return &Buffers{
		Trades:  NewTradeCh(ctx),
		Orders:  NewOrderCh(ctx),
		Markets: NewMarketCh(ctx),
	}
}
