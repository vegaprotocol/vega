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
	"context"
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type AccountSource interface {
	Obtain(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, id AccountID) (Account, error)
}

type _Transfer struct{}

type TransferID = ID[_Transfer]

type Transfer struct {
	VegaTime            time.Time
	DeliverOn           *time.Time
	Reason              *string
	DispatchMetricAsset *string
	DispatchMetric      *vega.DispatchMetric
	Factor              *decimal.Decimal
	EndEpoch            *uint64
	StartEpoch          *uint64
	ToAccountID         AccountID
	Reference           string
	Amount              decimal.Decimal
	AssetID             AssetID
	ID                  TransferID
	FromAccountID       AccountID
	TxHash              TxHash
	DispatchMarkets     []string
	TransferType        TransferType
	Status              TransferStatus
}

func (t *Transfer) ToProto(ctx context.Context, accountSource AccountSource) (*eventspb.Transfer, error) {
	fromAcc, err := accountSource.GetByID(ctx, t.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting from account for transfer proto:%w", err)
	}

	toAcc, err := accountSource.GetByID(ctx, t.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting to account for transfer proto:%w", err)
	}

	proto := eventspb.Transfer{
		Id:              t.ID.String(),
		From:            fromAcc.PartyID.String(),
		FromAccountType: fromAcc.Type,
		To:              toAcc.PartyID.String(),
		ToAccountType:   toAcc.Type,
		Asset:           t.AssetID.String(),
		Amount:          t.Amount.String(),
		Reference:       t.Reference,
		Status:          eventspb.Transfer_Status(t.Status),
		Timestamp:       t.VegaTime.UnixNano(),
		Kind:            nil,
		Reason:          t.Reason,
	}

	switch t.TransferType {
	case OneOff:
		proto.Kind = &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: t.DeliverOn.UnixNano()}}
	case Recurring:

		recurringTransfer := &eventspb.RecurringTransfer{
			StartEpoch: *t.StartEpoch,
			Factor:     t.Factor.String(),
		}
		if t.DispatchMetricAsset != nil {
			recurringTransfer.DispatchStrategy = &vega.DispatchStrategy{
				AssetForMetric: *t.DispatchMetricAsset,
				Metric:         *t.DispatchMetric,
				Markets:        t.DispatchMarkets,
			}
		}

		if t.EndEpoch != nil {
			endEpoch := *t.EndEpoch
			recurringTransfer.EndEpoch = &endEpoch
		}

		proto.Kind = &eventspb.Transfer_Recurring{Recurring: recurringTransfer}

	case Unknown:
		// leave Kind as nil
	}

	return &proto, nil
}

func TransferFromProto(ctx context.Context, t *eventspb.Transfer, txHash TxHash, vegaTime time.Time, accountSource AccountSource) (*Transfer, error) {
	fromAcc := Account{
		ID:       "",
		PartyID:  PartyID(t.From),
		AssetID:  AssetID(t.Asset),
		Type:     t.FromAccountType,
		TxHash:   txHash,
		VegaTime: time.Unix(0, t.Timestamp),
	}

	err := accountSource.Obtain(ctx, &fromAcc)
	if err != nil {
		return nil, fmt.Errorf("obtaining from account id for transfer:%w", err)
	}

	toAcc := Account{
		ID:       "",
		PartyID:  PartyID(t.To),
		AssetID:  AssetID(t.Asset),
		Type:     t.ToAccountType,
		TxHash:   txHash,
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
		ID:            TransferID(t.Id),
		TxHash:        txHash,
		VegaTime:      vegaTime,
		FromAccountID: fromAcc.ID,
		ToAccountID:   toAcc.ID,
		Amount:        amount,
		AssetID:       AssetID(t.Asset),
		Reference:     t.Reference,
		Status:        TransferStatus(t.Status),
		TransferType:  0,
		DeliverOn:     nil,
		StartEpoch:    nil,
		EndEpoch:      nil,
		Factor:        nil,
		Reason:        t.Reason,
	}

	switch v := t.Kind.(type) {
	case *eventspb.Transfer_OneOff:
		transfer.TransferType = OneOff
		if v.OneOff != nil {
			deliverOn := time.Unix(0, v.OneOff.DeliverOn)
			transfer.DeliverOn = &deliverOn
		}
	case *eventspb.Transfer_Recurring:
		transfer.TransferType = Recurring
		transfer.StartEpoch = &v.Recurring.StartEpoch
		if v.Recurring.DispatchStrategy != nil {
			transfer.DispatchMetric = &v.Recurring.DispatchStrategy.Metric
			transfer.DispatchMetricAsset = &v.Recurring.DispatchStrategy.AssetForMetric
			transfer.DispatchMarkets = v.Recurring.DispatchStrategy.Markets
		}

		if v.Recurring.EndEpoch != nil {
			endEpoch := *v.Recurring.EndEpoch
			transfer.EndEpoch = &endEpoch
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

func (t Transfer) Cursor() *Cursor {
	wc := TransferCursor{
		VegaTime: t.VegaTime,
		ID:       t.ID,
	}
	return NewCursor(wc.String())
}

func (t Transfer) ToProtoEdge(input ...any) (*v2.TransferEdge, error) {
	if len(input) != 2 {
		return nil, fmt.Errorf("expected account source and context argument")
	}

	ctx, ok := input[0].(context.Context)
	if !ok {
		return nil, fmt.Errorf("first argument must be a context.Context, got: %v", input[0])
	}

	as, ok := input[1].(AccountSource)
	if !ok {
		return nil, fmt.Errorf("second argument must be an AccountSource, got: %v", input[1])
	}

	transferProto, err := t.ToProto(ctx, as)
	if err != nil {
		return nil, err
	}
	return &v2.TransferEdge{
		Node:   transferProto,
		Cursor: t.Cursor().Encode(),
	}, nil
}

type TransferCursor struct {
	VegaTime time.Time  `json:"vegaTime"`
	ID       TransferID `json:"id"`
}

func (tc TransferCursor) String() string {
	bs, err := json.Marshal(tc)
	if err != nil {
		// This should never happen
		panic(fmt.Errorf("failed to marshal withdrawal cursor: %w", err))
	}
	return string(bs)
}

func (tc *TransferCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), tc)
}
