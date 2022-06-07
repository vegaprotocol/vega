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

var LedgerEntryColumns = []string{
	"account_from_id", "account_to_id", "quantity",
	"vega_time", "transfer_time", "reference", "type"}

func (le LedgerEntry) ToRow() []any {
	return []any{
		le.AccountFromID,
		le.AccountToID,
		le.Quantity,
		le.VegaTime,
		le.TransferTime,
		le.Reference,
		le.Type}
}
