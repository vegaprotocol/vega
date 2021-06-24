package num

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

func NewDecimalFromFloat(f float64) Decimal {
	return decimal.NewFromFloat(f)
}

func NewDecimalFromBigInt(value *big.Int, exp int32) Decimal {
	return decimal.NewFromBigInt(value, exp)
}

func DecimalFromUint(u *Uint) Decimal {
	return decimal.NewFromBigInt(u.BigInt(), 0)
}

func DecimalFromFloat(v float64) Decimal {
	return decimal.NewFromFloat(v)
}

func DecimalFromString(s string) (Decimal, error) {
	return decimal.NewFromString(s)
}

func MaxD(a, b Decimal) Decimal {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

func MinD(a, b Decimal) Decimal {
	if a.LessThan(b) {
		return a
	}
	return b
}
