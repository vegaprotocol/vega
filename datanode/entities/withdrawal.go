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
	"google.golang.org/protobuf/encoding/protojson"
)

type _Withdrawal struct{}

type WithdrawalID = ID[_Withdrawal]

type Withdrawal struct {
	ID                 WithdrawalID
	PartyID            PartyID
	Amount             decimal.Decimal
	Asset              AssetID
	Status             WithdrawalStatus
	Ref                string
	ForeignTxHash      string
	CreatedTimestamp   time.Time
	WithdrawnTimestamp time.Time
	Ext                WithdrawExt
	TxHash             TxHash
	VegaTime           time.Time
}

func WithdrawalFromProto(withdrawal *vega.Withdrawal, txHash TxHash, vegaTime time.Time) (*Withdrawal, error) {
	var err error
	var amount decimal.Decimal

	if amount, err = decimal.NewFromString(withdrawal.Amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	return &Withdrawal{
		ID:                 WithdrawalID(withdrawal.Id),
		PartyID:            PartyID(withdrawal.PartyId),
		Amount:             amount,
		Asset:              AssetID(withdrawal.Asset),
		Status:             WithdrawalStatus(withdrawal.Status),
		Ref:                withdrawal.Ref,
		ForeignTxHash:      withdrawal.TxHash,
		CreatedTimestamp:   NanosToPostgresTimestamp(withdrawal.CreatedTimestamp),
		WithdrawnTimestamp: NanosToPostgresTimestamp(withdrawal.WithdrawnTimestamp),
		Ext:                WithdrawExt{withdrawal.Ext},
		TxHash:             txHash,
		VegaTime:           vegaTime,
	}, nil
}

func (w Withdrawal) ToProto() *vega.Withdrawal {
	return &vega.Withdrawal{
		Id:                 w.ID.String(),
		PartyId:            w.PartyID.String(),
		Amount:             w.Amount.String(),
		Asset:              w.Asset.String(),
		Status:             vega.Withdrawal_Status(w.Status),
		Ref:                w.Ref,
		TxHash:             w.ForeignTxHash,
		CreatedTimestamp:   w.CreatedTimestamp.UnixNano(),
		WithdrawnTimestamp: w.WithdrawnTimestamp.UnixNano(),
		Ext:                w.Ext.WithdrawExt,
	}
}

func (w Withdrawal) Cursor() *Cursor {
	wc := WithdrawalCursor{
		VegaTime: w.VegaTime,
		ID:       w.ID,
	}
	return NewCursor(wc.String())
}

func (w Withdrawal) ToProtoEdge(_ ...any) (*v2.WithdrawalEdge, error) {
	return &v2.WithdrawalEdge{
		Node:   w.ToProto(),
		Cursor: w.Cursor().Encode(),
	}, nil
}

type WithdrawExt struct {
	*vega.WithdrawExt
}

func (we WithdrawExt) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(we)
}

func (we *WithdrawExt) UnmarshalJSON(b []byte) error {
	we.WithdrawExt = &vega.WithdrawExt{}
	return protojson.Unmarshal(b, we)
}

type WithdrawalCursor struct {
	VegaTime time.Time    `json:"vegaTime"`
	ID       WithdrawalID `json:"id"`
}

func (wc WithdrawalCursor) String() string {
	bs, err := json.Marshal(wc)
	if err != nil {
		// This should never happen
		panic(fmt.Errorf("failed to marshal withdrawal cursor: %w", err))
	}
	return string(bs)
}

func (wc *WithdrawalCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), wc)
}
