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
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	VegaTime time.Time
	*Account
	Balance decimal.Decimal
	TxHash  TxHash
}

func (ab AccountBalance) ToProto() *v2.AccountBalance {
	return &v2.AccountBalance{
		Owner:    ab.PartyID.String(),
		Balance:  ab.Balance.String(),
		Asset:    ab.AssetID.String(),
		MarketId: ab.MarketID.String(),
		Type:     ab.Account.Type,
	}
}

func (ab AccountBalance) ToProtoEdge(_ ...any) (*v2.AccountEdge, error) {
	return &v2.AccountEdge{
		Node:   ab.ToProto(),
		Cursor: ab.Cursor().Encode(),
	}, nil
}

type AccountBalanceKey struct {
	VegaTime  time.Time
	AccountID AccountID
}

func (ab AccountBalance) Key() AccountBalanceKey {
	return AccountBalanceKey{ab.Account.ID, ab.VegaTime}
}

func (ab AccountBalance) ToRow() []interface{} {
	return []interface{}{ab.Account.ID, ab.TxHash, ab.VegaTime, ab.Balance}
}

func (ab AccountBalance) Cursor() *Cursor {
	cursor := AccountCursor{
		AccountID: ab.Account.ID,
	}

	return NewCursor(cursor.String())
}

func (ab AccountBalance) Equal(other AccountBalance) bool {
	return ab.AssetID == other.AssetID &&
		ab.PartyID == other.PartyID &&
		ab.MarketID == other.MarketID &&
		ab.Type == other.Type &&
		ab.Balance.Equal(other.Balance)
}

type AccountCursor struct {
	AccountID AccountID `json:"account_id"`
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
