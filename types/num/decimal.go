package num

import (
	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

func NewDecimalFromFloat(f float64) Decimal {
	return decimal.NewFromFloat(f)
}

func DecimalFromUint(u *Uint) Decimal {
	return decimal.NewFromBigInt(u.u.ToBig(), 0)
}
