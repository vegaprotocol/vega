// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
