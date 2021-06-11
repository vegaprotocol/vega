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
