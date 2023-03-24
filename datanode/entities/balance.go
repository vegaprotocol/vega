// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

type Balance struct {
	AccountID AccountID
	TxHash    TxHash
	VegaTime  time.Time
	Balance   decimal.Decimal
}

type BalanceKey struct {
	VegaTime  time.Time
	AccountID AccountID
}

func (b Balance) Key() BalanceKey {
	return BalanceKey{b.AccountID, b.VegaTime}
}

var BalanceColumns = []string{"account_id", "tx_hash", "vega_time", "balance"}

func (b Balance) ToRow() []interface{} {
	return []interface{}{b.AccountID, b.TxHash, b.VegaTime, b.Balance}
}
