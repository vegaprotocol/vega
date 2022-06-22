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

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	*Account
	Balance  decimal.Decimal
	VegaTime time.Time
}

func (ab *AccountBalance) ToProto() *vega.Account {
	return &vega.Account{
		Owner:    ab.PartyID.String(),
		Balance:  ab.Balance.String(),
		Asset:    ab.AssetID.String(),
		MarketId: ab.MarketID.String(),
		Type:     ab.Account.Type,
	}
}

type AccountBalanceKey struct {
	AccountID int64
	VegaTime  time.Time
}

func (b AccountBalance) Key() AccountBalanceKey {
	return AccountBalanceKey{b.Account.ID, b.VegaTime}
}

func (b AccountBalance) ToRow() []interface{} {
	return []interface{}{b.Account.ID, b.VegaTime, b.Balance}
}
