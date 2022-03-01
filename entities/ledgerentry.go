package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type LedgerEntry struct {
	ID            int64
	AccountFromID int64
	AccountToID   int64
	Quantity      decimal.Decimal
	VegaTime      time.Time
	TransferTime  time.Time
	Reference     string
	Type          string
}
