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
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
)

type (
	_Account  struct{}
	AccountID = ID[_Account]
)

type Account struct {
	ID       AccountID
	PartyID  PartyID
	AssetID  AssetID
	MarketID MarketID
	Type     vega.AccountType
	TxHash   TxHash
	VegaTime time.Time
}

func (a Account) ToProto() *vega.Account {
	return &vega.Account{
		Id:       a.ID.String(),
		Owner:    a.PartyID.String(),
		Asset:    a.AssetID.String(),
		MarketId: a.MarketID.String(),
		Type:     a.Type,
	}
}

func (a Account) ToAccountDetailsProto() *vega.AccountDetails {
	return &vega.AccountDetails{
		Owner:    ptr.From(a.PartyID.String()),
		AssetId:  a.AssetID.String(),
		MarketId: ptr.From(a.MarketID.String()),
		Type:     a.Type,
	}
}

func (a Account) String() string {
	return fmt.Sprintf("{ID: %s}", a.ID)
}

func AccountFromProto(va *vega.Account, txHash TxHash) (Account, error) {
	// In account proto messages, network party is '*' and no market is '!'
	partyID := va.Owner
	if partyID == "*" {
		partyID = "network"
	}

	marketID := va.MarketId
	if marketID == "!" {
		marketID = ""
	}

	account := Account{
		PartyID:  PartyID(partyID),
		AssetID:  AssetID(va.Asset),
		MarketID: MarketID(marketID),
		Type:     va.Type,
		TxHash:   txHash,
	}
	return account, nil
}

func AccountProtoFromDetails(ad *vega.AccountDetails, txHash TxHash) (Account, error) {
	marketID, partyID := "", "network"
	if ad.MarketId != nil {
		marketID = *ad.MarketId
	}
	if ad.Owner != nil {
		partyID = *ad.Owner
	}
	return Account{
		TxHash:   txHash,
		PartyID:  ID[_Party](partyID),
		MarketID: ID[_Market](marketID),
		Type:     ad.Type,
		AssetID:  ID[_Asset](ad.AssetId),
	}, nil
}
