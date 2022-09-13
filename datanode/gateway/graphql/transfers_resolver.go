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

package gql

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

var ErrUnsupportedTransferKind = errors.New("unsupported transfer kind")

type transferInstructionResolver VegaResolverRoot

func (r *transferInstructionResolver) Asset(ctx context.Context, obj *eventspb.TransferInstruction) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *transferInstructionResolver) Timestamp(ctx context.Context, obj *eventspb.TransferInstruction) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

func (r *transferInstructionResolver) Kind(ctx context.Context, obj *eventspb.TransferInstruction) (TransferInstructionKind, error) {
	switch obj.GetKind().(type) {
	case *eventspb.TransferInstruction_OneOff:
		return obj.GetOneOff(), nil
	case *eventspb.TransferInstruction_Recurring:
		return obj.GetRecurring(), nil
	default:
		return nil, ErrUnsupportedTransferKind
	}
}

type recurringTransferInstructionResolver VegaResolverRoot

func (r *recurringTransferInstructionResolver) StartEpoch(ctx context.Context, obj *eventspb.RecurringTransferInstruction) (int, error) {
	return int(obj.StartEpoch), nil
}

func (r *recurringTransferInstructionResolver) EndEpoch(ctx context.Context, obj *eventspb.RecurringTransferInstruction) (*int, error) {
	if obj.EndEpoch != nil {
		i := int(*obj.EndEpoch)
		return &i, nil
	}
	return nil, nil
}

func (r *recurringTransferInstructionResolver) DispatchStrategy(ctx context.Context, obj *eventspb.RecurringTransferInstruction) (*DispatchStrategy, error) {
	if obj.DispatchStrategy != nil {
		return &DispatchStrategy{
			DispatchMetric:        obj.DispatchStrategy.Metric,
			DispatchMetricAssetID: obj.DispatchStrategy.AssetForMetric,
			MarketIdsInScope:      obj.DispatchStrategy.Markets,
		}, nil
	}
	return nil, nil
}

type oneoffTransferInstructionResolver VegaResolverRoot

func (r *oneoffTransferInstructionResolver) DeliverOn(ctx context.Context, obj *eventspb.OneOffTransferInstruction) (*string, error) {
	if obj.DeliverOn > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.DeliverOn))
		return &t, nil
	}
	return nil, nil
}
