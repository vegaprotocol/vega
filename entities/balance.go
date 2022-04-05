package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balance struct {
	AccountID int64
	VegaTime  time.Time
	Balance   decimal.Decimal
}

type BalanceKey struct {
	AccountID int64
	VegaTime  time.Time
}

func (b Balance) Key() BalanceKey {
	return BalanceKey{b.AccountID, b.VegaTime}
}

var BalanceColumns = []string{"account_id", "vega_time", "balance"}

func (b Balance) ToRow() []interface{} {
	return []interface{}{b.AccountID, b.VegaTime, b.Balance}
}
