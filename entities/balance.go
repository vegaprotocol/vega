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
