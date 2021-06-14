package num

import (
	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

func NewDecimalFromFloat(f float64) Decimal {
	return decimal.NewFromFloat(f)
}
