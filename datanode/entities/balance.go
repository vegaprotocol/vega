// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balance struct {
	AccountID AccountID
	TxHash    TxHash
	VegaTime  time.Time
	Balance   decimal.Decimal
}

type BalanceKey struct {
	AccountID AccountID
	VegaTime  time.Time
}

func (b Balance) Key() BalanceKey {
	return BalanceKey{b.AccountID, b.VegaTime}
}

var BalanceColumns = []string{"account_id", "tx_hash", "vega_time", "balance"}

func (b Balance) ToRow() []interface{} {
	return []interface{}{b.AccountID, b.TxHash, b.VegaTime, b.Balance}
}
