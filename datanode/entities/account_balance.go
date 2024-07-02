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
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	*Account
	Balance  decimal.Decimal
	TxHash   TxHash
	VegaTime time.Time
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

func (ab AccountBalance) ToProtoWithParent(parentPartyID *string) *v2.AccountBalance {
	proto := ab.ToProto()
	proto.ParentPartyId = parentPartyID
	return proto
}

func (ab AccountBalance) ToProtoEdge(args ...any) (*v2.AccountEdge, error) {
	var parentPartyID *string
	if len(args) > 0 {
		perPartyDerivedKey, ok := args[0].(map[string]string)
		if !ok {
			return nil, fmt.Errorf("expected argument of type map[string]string, got %T", args[0])
		}

		if party, isDerivedKey := perPartyDerivedKey[ab.PartyID.String()]; isDerivedKey {
			parentPartyID = &party
		}
	}

	return &v2.AccountEdge{
		Node:   ab.ToProtoWithParent(parentPartyID),
		Cursor: ab.Cursor().Encode(),
	}, nil
}

type AccountBalanceKey struct {
	AccountID AccountID
	VegaTime  time.Time
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
