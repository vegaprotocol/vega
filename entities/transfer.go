package entities

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type AccountSource interface {
	Obtain(ctx context.Context, a *Account) error
	GetByID(id int64) (Account, error)
}

type TransferID struct{ ID }

func NewTransferID(id string) TransferID {
	return TransferID{ID: ID(id)}
}

type Transfer struct {
	ID            TransferID
	VegaTime      time.Time
	FromAccountId int64
	ToAccountId   int64
	AssetId       AssetID
	MarketId      MarketID
	Amount        decimal.Decimal
	Reference     string
	Status        TransferStatus
	TransferType  TransferType
	DeliverOn     *time.Time
	StartEpoch    *uint64
	EndEpoch      *uint64
	Factor        *decimal.Decimal
}

func (t *Transfer) ToProto(accountSource AccountSource) (*eventspb.Transfer, error) {

	fromAcc, err := accountSource.GetByID(t.FromAccountId)
	if err != nil {
		return nil, fmt.Errorf("getting from account for transfer proto:%w", err)
	}

	toAcc, err := accountSource.GetByID(t.ToAccountId)
	if err != nil {
		return nil, fmt.Errorf("getting to account for transfer proto:%w", err)
	}

	proto := eventspb.Transfer{
		Id:              t.ID.String(),
		From:            fromAcc.PartyID.String(),
		FromAccountType: fromAcc.Type,
		To:              toAcc.PartyID.String(),
		ToAccountType:   toAcc.Type,
		Asset:           t.AssetId.String(),
		Market:          t.MarketId.String(),
		Amount:          t.Amount.String(),
		Reference:       t.Reference,
		Status:          eventspb.Transfer_Status(t.Status),
		Timestamp:       t.VegaTime.UnixNano(),
		Kind:            nil,
	}

	switch t.TransferType {
	case OneOff:
		proto.Kind = &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: t.DeliverOn.Unix()}}
	case Recurring:
		recurringTransfer := &eventspb.RecurringTransfer{
			StartEpoch: *t.StartEpoch,
			Factor:     t.Factor.String(),
		}

		if t.EndEpoch != nil {
			recurringTransfer.EndEpoch = &vega.Uint64Value{Value: *t.EndEpoch}
		}

		proto.Kind = &eventspb.Transfer_Recurring{Recurring: recurringTransfer}

	case Unknown:
		// leave Kind as nil
	}

	return &proto, nil
}

func TransferFromProto(ctx context.Context, t *eventspb.Transfer, vegaTime time.Time, accountSource AccountSource) (*Transfer, error) {

	fromAcc := Account{
		ID:       0,
		PartyID:  PartyID{ID(t.From)},
		AssetID:  AssetID{ID(t.Asset)},
		MarketID: MarketID{},
		Type:     t.FromAccountType,
		VegaTime: vegaTime,
	}

	err := accountSource.Obtain(ctx, &fromAcc)

	if err != nil {
		return nil, fmt.Errorf("obtaining from account id for transfer:%w", err)
	}

	toAcc := Account{
		ID:       0,
		PartyID:  PartyID{ID: ID(t.To)},
		AssetID:  AssetID{ID: ID(t.Asset)},
		MarketID: MarketID{ID: ID(t.Market)},
		Type:     t.ToAccountType,
		VegaTime: vegaTime,
	}

	err = accountSource.Obtain(ctx, &toAcc)

	if err != nil {
		return nil, fmt.Errorf("obtaining to account id for transfer:%w", err)
	}

	amount, err := decimal.NewFromString(t.Amount)
	if err != nil {
		return nil, fmt.Errorf("getting amount for transfer:%w", err)
	}

	transfer := Transfer{
		ID:            NewTransferID(t.Id),
		VegaTime:      vegaTime,
		FromAccountId: fromAcc.ID,
		ToAccountId:   toAcc.ID,
		Amount:        amount,
		AssetId:       NewAssetID(t.Asset),
		MarketId:      NewMarketID(t.Market),
		Reference:     t.Reference,
		Status:        TransferStatus(t.Status),
		TransferType:  0,
		DeliverOn:     nil,
		StartEpoch:    nil,
		EndEpoch:      nil,
		Factor:        nil,
	}

	switch v := t.Kind.(type) {
	case *eventspb.Transfer_OneOff:
		transfer.TransferType = OneOff
		deliverOn := time.Unix(v.OneOff.DeliverOn, 0)
		transfer.DeliverOn = &deliverOn
	case *eventspb.Transfer_Recurring:
		transfer.TransferType = Recurring
		transfer.StartEpoch = &v.Recurring.StartEpoch
		if v.Recurring.EndEpoch != nil {
			transfer.EndEpoch = &v.Recurring.EndEpoch.Value
		}
		factor, err := decimal.NewFromString(v.Recurring.Factor)
		if err != nil {
			return nil, fmt.Errorf("getting factor for transfer:%w", err)
		}

		transfer.Factor = &factor
	default:
		transfer.TransferType = Unknown
	}

	return &transfer, nil
}
