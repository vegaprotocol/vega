package entities

import (
	"fmt"
	"time"

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
		CreatedTimestamp:   time.Unix(0, withdrawal.CreatedTimestamp),
		WithdrawnTimestamp: time.Unix(0, withdrawal.WithdrawnTimestamp),
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
