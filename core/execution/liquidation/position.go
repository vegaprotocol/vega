package liquidation

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type Pos struct {
	open int64
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
	return num.UintZero() // shouldn't be used
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
	return num.UintZero() // shouldn't be used
}
