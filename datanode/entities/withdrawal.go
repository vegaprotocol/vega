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
	"google.golang.org/protobuf/encoding/protojson"
)

type WithdrawalID struct{ ID }

func NewWithdrawalID(id string) WithdrawalID {
	return WithdrawalID{ID: ID(id)}
}

type Withdrawal struct {
	ID                 WithdrawalID
	PartyID            PartyID
	Amount             decimal.Decimal
	Asset              AssetID
	Status             WithdrawalStatus
	Ref                string
	Expiry             time.Time
	TxHash             string
	CreatedTimestamp   time.Time
	WithdrawnTimestamp time.Time
	Ext                WithdrawExt
	VegaTime           time.Time
}

func WithdrawalFromProto(withdrawal *vega.Withdrawal, vegaTime time.Time) (*Withdrawal, error) {
	var err error
	var amount decimal.Decimal

	if amount, err = decimal.NewFromString(withdrawal.Amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	return &Withdrawal{
		ID:      NewWithdrawalID(withdrawal.Id),
		PartyID: NewPartyID(withdrawal.PartyId),
		Amount:  amount,
		Asset:   NewAssetID(withdrawal.Asset),
		Status:  WithdrawalStatus(withdrawal.Status),
		Ref:     withdrawal.Ref,
		// According to the GraphQL resolver, the expiry is the Unix time, not UnixNano
		Expiry:             time.Unix(withdrawal.Expiry, 0),
		TxHash:             withdrawal.TxHash,
		CreatedTimestamp:   NanosToPostgresTimestamp(withdrawal.CreatedTimestamp),
		WithdrawnTimestamp: NanosToPostgresTimestamp(withdrawal.WithdrawnTimestamp),
		Ext:                WithdrawExt{withdrawal.Ext},
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
		Expiry:             w.Expiry.Unix(),
		TxHash:             w.TxHash,
		CreatedTimestamp:   w.CreatedTimestamp.UnixNano(),
		WithdrawnTimestamp: w.WithdrawnTimestamp.UnixNano(),
		Ext:                w.Ext.WithdrawExt,
	}
}

func (w Withdrawal) Cursor() *Cursor {
	wc := WithdrawalCursor{
		VegaTime: w.VegaTime,
		ID:       w.ID.String(),
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
	VegaTime time.Time `json:"vegaTime"`
	ID       string    `json:"id"`
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
