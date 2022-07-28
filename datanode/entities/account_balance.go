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
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	*Account
	Balance  decimal.Decimal
	VegaTime time.Time
}

func (ab *AccountBalance) ToProto() *vega.Account {
	marketId := ab.MarketID.String()
	if marketId == noMarketStr {
		marketId = ""
	}

	ownerId := ab.PartyID.String()
	if ownerId == systemOwnerStr {
		ownerId = ""
	}

	return &vega.Account{
		Owner:    ownerId,
		Balance:  ab.Balance.String(),
		Asset:    ab.AssetID.String(),
		MarketId: marketId,
		Type:     ab.Account.Type,
	}
}

func (ab AccountBalance) ToProtoEdge(_ ...any) (*v2.AccountEdge, error) {
	return &v2.AccountEdge{
		Account: ab.ToProto(),
		Cursor:  ab.Cursor().Encode(),
	}, nil
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

func (b AccountBalance) Cursor() *Cursor {
	cursor := AccountCursor{
		AccountID: b.Account.ID,
	}

	return NewCursor(cursor.String())
}

func (b AccountBalance) Equal(other AccountBalance) bool {
	return b.AssetID == other.AssetID &&
		b.PartyID == other.PartyID &&
		b.MarketID == other.MarketID &&
		b.Type == other.Type &&
		b.Balance.Equal(other.Balance)
}

type AccountCursor struct {
	AccountID int64 `json:"account_id"`
}

func (ac AccountCursor) String() string {
	bs, err := json.Marshal(ac)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("could not marshal account cursor: %w", err))
	}
	return string(bs)
}

func (ac *AccountCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), ac)
}
