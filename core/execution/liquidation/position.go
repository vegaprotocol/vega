package liquidation

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type Pos struct {
	open  int64
	price *num.Uint
}

func (p *Pos) Party() string {
	return types.NetworkParty
}

func (p *Pos) Size() int64 {
	return p.open
}

func (p *Pos) Buy() int64 {
	return 0
}

func (p *Pos) Sell() int64 {
	return 0
}

func (p *Pos) Price() *num.Uint {
	if p.price == nil {
		return num.UintZero()
	}
	return p.price.Clone()
}

func (p *Pos) BuySumProduct() *num.Uint {
	return num.UintZero() // shouldn't be used
}

func (p *Pos) SellSumProduct() *num.Uint {
	return num.UintZero() // shouldn't be used
}

func (p *Pos) VWBuy() *num.Uint {
	return num.UintZero() // shouldn't be used
}

func (p *Pos) VWSell() *num.Uint {
	return num.UintZero() // shouldn't be used
}

func (p *Pos) AverageEntryPrice() *num.Uint {
	if p.price != nil {
		return p.price.Clone() // not sure
	}
	return num.UintZero() // shouldn't be used
}
