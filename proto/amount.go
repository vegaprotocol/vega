package proto

import (
	"github.com/shopspring/decimal"
)

// Decimal - single function to automatically convert the gRPC string values to the decimal type
func (a Amount) Decimal() (decimal.Decimal, error) {
	return decimal.NewFromString(a.Value)
}
