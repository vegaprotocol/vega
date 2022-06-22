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
	ID                  TransferID
	VegaTime            time.Time
	FromAccountId       int64
	ToAccountId         int64
	AssetId             AssetID
	Amount              decimal.Decimal
	Reference           string
	Status              TransferStatus
	TransferType        TransferType
	DeliverOn           *time.Time
	StartEpoch          *uint64
	EndEpoch            *uint64
	Factor              *decimal.Decimal
	DispatchMetric      *vega.DispatchMetric
	DispatchMetricAsset *string
	DispatchMarkets     []string
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
		if t.DispatchMetricAsset != nil {
			recurringTransfer.DispatchStrategy = &vega.DispatchStrategy{
				AssetForMetric: *t.DispatchMetricAsset,
				Metric:         vega.DispatchMetric(*t.DispatchMetric),
				Markets:        t.DispatchMarkets,
			}
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
		if v.Recurring.DispatchStrategy != nil {
			transfer.DispatchMetric = &v.Recurring.DispatchStrategy.Metric
			transfer.DispatchMetricAsset = &v.Recurring.DispatchStrategy.AssetForMetric
			transfer.DispatchMarkets = v.Recurring.DispatchStrategy.Markets
		}

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
