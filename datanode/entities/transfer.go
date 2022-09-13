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
	GetByID(id int64) (Account, error)
}

type _TransferInstruction struct{}

type TransferInstructionID = ID[_TransferInstruction]

type TransferInstruction struct {
	ID                  TransferInstructionID
	TxHash              TxHash
	VegaTime            time.Time
	FromAccountID       int64
	ToAccountID         int64
	AssetID             AssetID
	Amount              decimal.Decimal
	Reference           string
	Status              TransferInstructionStatus
	TransferInstructionType        TransferInstructionType
	DeliverOn           *time.Time
	StartEpoch          *uint64
	EndEpoch            *uint64
	Factor              *decimal.Decimal
	DispatchMetric      *vega.DispatchMetric
	DispatchMetricAsset *string
	DispatchMarkets     []string
}

func (t *TransferInstruction) ToProto(accountSource AccountSource) (*eventspb.TransferInstruction, error) {
	fromAcc, err := accountSource.GetByID(t.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting from account for transfer proto:%w", err)
	}

	toAcc, err := accountSource.GetByID(t.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting to account for transfer proto:%w", err)
	}

	proto := eventspb.TransferInstruction{
		Id:              t.ID.String(),
		From:            fromAcc.PartyID.String(),
		FromAccountType: fromAcc.Type,
		To:              toAcc.PartyID.String(),
		ToAccountType:   toAcc.Type,
		Asset:           t.AssetID.String(),
		Amount:          t.Amount.String(),
		Reference:       t.Reference,
		Status:          eventspb.TransferInstruction_Status(t.Status),
		Timestamp:       t.VegaTime.UnixNano(),
		Kind:            nil,
	}

	switch t.TransferInstructionType {
	case OneOff:
		proto.Kind = &eventspb.TransferInstruction_OneOff{OneOff: &eventspb.OneOffTransferInstruction{DeliverOn: t.DeliverOn.Unix()}}
	case Recurring:

		recurringTransfer := &eventspb.RecurringTransferInstruction{
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

		proto.Kind = &eventspb.TransferInstruction_Recurring{Recurring: recurringTransfer}

	case Unknown:
		// leave Kind as nil
	}

	return &proto, nil
}

func TransferInstructionFromProto(ctx context.Context, t *eventspb.TransferInstruction, txHash TxHash, vegaTime time.Time, accountSource AccountSource) (*TransferInstruction, error) {
	fromAcc := Account{
		ID:       0,
		PartyID:  PartyID(t.From),
		AssetID:  AssetID(t.Asset),
		Type:     t.FromAccountType,
		TxHash:   txHash,
		VegaTime: vegaTime,
	}

	err := accountSource.Obtain(ctx, &fromAcc)
	if err != nil {
		return nil, fmt.Errorf("obtaining from account id for transfer:%w", err)
	}

	toAcc := Account{
		ID:       0,
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

	transfer := TransferInstruction{
		ID:            TransferInstructionID(t.Id),
		TxHash:        txHash,
		VegaTime:      vegaTime,
		FromAccountID: fromAcc.ID,
		ToAccountID:   toAcc.ID,
		Amount:        amount,
		AssetID:       AssetID(t.Asset),
		Reference:     t.Reference,
		Status:        TransferInstructionStatus(t.Status),
		TransferInstructionType:  0,
		DeliverOn:     nil,
		StartEpoch:    nil,
		EndEpoch:      nil,
		Factor:        nil,
	}

	switch v := t.Kind.(type) {
	case *eventspb.TransferInstruction_OneOff:
		transfer.TransferInstructionType = OneOff
		deliverOn := time.Unix(v.OneOff.DeliverOn, 0)
		transfer.DeliverOn = &deliverOn
	case *eventspb.TransferInstruction_Recurring:
		transfer.TransferInstructionType = Recurring
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
		transfer.TransferInstructionType = Unknown
	}

	return &transfer, nil
}

func (t TransferInstruction) Cursor() *Cursor {
	wc := TransferInstructionCursor{
		VegaTime: t.VegaTime,
		ID:       t.ID,
	}
	return NewCursor(wc.String())
}

func (t TransferInstruction) ToProtoEdge(input ...any) (*v2.TransferInstructionEdge, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("expected account source argument")
	}

	switch as := input[0].(type) {
	case AccountSource:
		transferProto, err := t.ToProto(as)
		if err != nil {
			return nil, err
		}
		return &v2.TransferInstructionEdge{
			Node:   transferProto,
			Cursor: t.Cursor().Encode(),
		}, nil
	default:
		return nil, fmt.Errorf("expected account source argument, got:%v", as)
	}
}

type TransferInstructionCursor struct {
	VegaTime time.Time  `json:"vegaTime"`
	ID       TransferInstructionID `json:"id"`
}

func (tc TransferInstructionCursor) String() string {
	bs, err := json.Marshal(tc)
	if err != nil {
		// This should never happen
		panic(fmt.Errorf("failed to marshal withdrawal cursor: %w", err))
	}
	return string(bs)
}

func (tc *TransferInstructionCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), tc)
}
