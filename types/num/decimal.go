package num

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

func NewDecimalFromBigInt(value *big.Int, exp int32) Decimal {
	return decimal.NewFromBigInt(value, exp)
}

func DecimalFromUint(u *Uint) Decimal {
	return decimal.NewFromBigInt(u.BigInt(), 0)
}
