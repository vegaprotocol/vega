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
	Expiry             time.Time
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
	ext := WithdrawExt{}
	if withdrawal.Ext != nil {
		cpy := *withdrawal.Ext
		// if erc20 := withdrawal.Ext.GetErc20(); erc20 != nil {
		// cpy20 := *erc20
		// cpy.Ext = &cpy20
		// }
		ext.WithdrawExt = &cpy
	}

	return &Withdrawal{
		ID:      WithdrawalID(withdrawal.Id),
		PartyID: PartyID(withdrawal.PartyId),
		Amount:  amount,
		Asset:   AssetID(withdrawal.Asset),
		Status:  WithdrawalStatus(withdrawal.Status),
		Ref:     withdrawal.Ref,
		// According to the GraphQL resolver, the expiry is the Unix time, not UnixNano
		Expiry:             time.Unix(withdrawal.Expiry, 0),
		ForeignTxHash:      withdrawal.TxHash,
		CreatedTimestamp:   NanosToPostgresTimestamp(withdrawal.CreatedTimestamp),
		WithdrawnTimestamp: NanosToPostgresTimestamp(withdrawal.WithdrawnTimestamp),
		Ext:                ext,
		TxHash:             txHash,
		VegaTime:           vegaTime,
	}, nil
}

func (w Withdrawal) ToProto() *vega.Withdrawal {
	var pbExt *vega.WithdrawExt
	if w.Ext.WithdrawExt != nil {
		cpy := *w.Ext.WithdrawExt
		// if erc20 := w.Ext.WithdrawExt.GetErc20(); erc20 != nil {
		// cpy20 := *erc20
		// cpy.Ext = &cpy20
		// }
		pbExt = &cpy
	}
	return &vega.Withdrawal{
		Id:                 w.ID.String(),
		PartyId:            w.PartyID.String(),
		Amount:             w.Amount.String(),
		Asset:              w.Asset.String(),
		Status:             vega.Withdrawal_Status(w.Status),
		Ref:                w.Ref,
		Expiry:             w.Expiry.Unix(),
		TxHash:             w.ForeignTxHash,
		CreatedTimestamp:   w.CreatedTimestamp.UnixNano(),
		WithdrawnTimestamp: w.WithdrawnTimestamp.UnixNano(),
		Ext:                pbExt,
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
