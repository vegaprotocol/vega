package settlement

type pos struct {
	party string
	size  int64
	price uint64
}

// Party - part of the MarketPosition interface, used to update position after SettlePreTrade
func (p pos) Party() string {
	return p.party
}

// Size - part of the MarketPosition interface, used to update position after SettlePreTrade
func (p pos) Size() int64 {
	return p.size
}

// Price - part of the MarketPosition interface, used to update position after SettlePreTrade
func (p pos) Price() uint64 {
	return p.price
}
