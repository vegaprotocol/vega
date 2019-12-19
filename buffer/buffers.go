package buffer

import "context"

type Buffers struct {
	Trades    *TradeCh
	Orders    *OrderCh
	Markets   *MarketCh
	Positions *Settlement
}

func New(ctx context.Context) *Buffers {
	return &Buffers{
		Trades:    NewTradeCh(ctx),
		Orders:    NewOrderCh(ctx),
		Markets:   NewMarketCh(ctx),
		Positions: NewSettlement(ctx),
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

func (b *Buffers) PositionsSub(buf int) *SettleSub {
	return b.Positions.Subscribe(buf)
}
