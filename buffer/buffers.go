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

func (b *Buffers) TradesSub(buf int) TradeSub {
	return b.Trades.Subscribe(buf)
}

func (b *Buffers) OrdersSub(buf int) OrderSub {
	return b.Orders.Subscribe(buf)
}

func (b *Buffers) MarketsSub(buf int) MarketSub {
	return b.Markets.Subscribe(buf)
}
