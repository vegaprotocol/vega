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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type _Deposit struct{}

type DepositID = ID[_Deposit]

type Deposit struct {
	ID                DepositID
	Status            DepositStatus
	PartyID           PartyID
	Asset             AssetID
	Amount            decimal.Decimal
	ForeignTxHash     string
	CreditedTimestamp time.Time
	CreatedTimestamp  time.Time
	TxHash            TxHash
	VegaTime          time.Time
}

func DepositFromProto(deposit *vega.Deposit, txHash TxHash, vegaTime time.Time) (*Deposit, error) {
	var err error
	var amount decimal.Decimal

	if amount, err = decimal.NewFromString(deposit.Amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	return &Deposit{
		ID:                DepositID(deposit.Id),
		Status:            DepositStatus(deposit.Status),
		PartyID:           PartyID(deposit.PartyId),
		Asset:             AssetID(deposit.Asset),
		Amount:            amount,
		ForeignTxHash:     deposit.TxHash,
		CreditedTimestamp: NanosToPostgresTimestamp(deposit.CreditedTimestamp),
		CreatedTimestamp:  NanosToPostgresTimestamp(deposit.CreatedTimestamp),
		TxHash:            txHash,
		VegaTime:          vegaTime,
	}, nil
}

func (d Deposit) ToProto() *vega.Deposit {
	return &vega.Deposit{
		Id:                d.ID.String(),
		Status:            vega.Deposit_Status(d.Status),
		PartyId:           d.PartyID.String(),
		Asset:             d.Asset.String(),
		Amount:            d.Amount.String(),
		TxHash:            d.ForeignTxHash,
		CreditedTimestamp: d.CreditedTimestamp.UnixNano(),
		CreatedTimestamp:  d.CreatedTimestamp.UnixNano(),
	}
}

func (d Deposit) Cursor() *Cursor {
	cursor := DepositCursor{
		VegaTime: d.VegaTime,
		ID:       d.ID,
	}
	return NewCursor(cursor.String())
}

func (d Deposit) ToProtoEdge(_ ...any) (*v2.DepositEdge, error) {
	return &v2.DepositEdge{
		Node:   d.ToProto(),
		Cursor: d.Cursor().Encode(),
	}, nil
}

type DepositCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	ID       DepositID `json:"id"`
}

func (dc DepositCursor) String() string {
	bs, err := json.Marshal(dc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("couldn't marshal deposit cursor: %w", err))
	}
	return string(bs)
}

func (dc *DepositCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), dc)
}
