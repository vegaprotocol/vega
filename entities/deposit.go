package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type DepositID struct{ ID }

func NewDepositID(id string) DepositID {
	return DepositID{ID: ID(id)}
}

type Deposit struct {
	ID                DepositID
	Status            DepositStatus
	PartyID           PartyID
	Asset             AssetID
	Amount            decimal.Decimal
	TxHash            string
	CreditedTimestamp time.Time
	CreatedTimestamp  time.Time
	VegaTime          time.Time
}

func DepositFromProto(deposit *vega.Deposit, vegaTime time.Time) (*Deposit, error) {
	var err error
	var amount decimal.Decimal

	if amount, err = decimal.NewFromString(deposit.Amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	return &Deposit{
		ID:                NewDepositID(deposit.Id),
		Status:            DepositStatus(deposit.Status),
		PartyID:           NewPartyID(deposit.PartyId),
		Asset:             NewAssetID(deposit.Asset),
		Amount:            amount,
		TxHash:            deposit.TxHash,
		CreditedTimestamp: time.Unix(0, deposit.CreditedTimestamp),
		CreatedTimestamp:  time.Unix(0, deposit.CreatedTimestamp),
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
		TxHash:            d.TxHash,
		CreditedTimestamp: d.CreditedTimestamp.UnixNano(),
		CreatedTimestamp:  d.CreatedTimestamp.UnixNano(),
	}
}
