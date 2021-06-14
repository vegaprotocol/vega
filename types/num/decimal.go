package num

import (
	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

func NewDecimalFromFloat(f float64) Decimal {
	return decimal.NewFromFloat(f)
}

func DivDec(a, b *Uint) Decimal {
	aDec := decimal.NewFromBigInt(a.u.ToBig(), 0)
	bDec := decimal.NewFromBigInt(b.u.ToBig(), 0)

	return aDec.Div(bDec)
}

func MulDec(a *Uint, b Decimal) Decimal {
	aDec := decimal.NewFromBigInt(a.u.ToBig(), 0)

	return b.Mul(aDec)
}
