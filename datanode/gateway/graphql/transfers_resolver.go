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

type transferResolver VegaResolverRoot

func (r *transferResolver) Asset(ctx context.Context, obj *eventspb.Transfer) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *transferResolver) Timestamp(ctx context.Context, obj *eventspb.Transfer) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

func (r *transferResolver) Kind(ctx context.Context, obj *eventspb.Transfer) (TransferKind, error) {
	switch obj.GetKind().(type) {
	case *eventspb.Transfer_OneOff:
		return obj.GetOneOff(), nil
	case *eventspb.Transfer_OneOffGovernance:
		return obj.GetOneOffGovernance(), nil
	case *eventspb.Transfer_Recurring:
		return obj.GetRecurring(), nil
	case *eventspb.Transfer_RecurringGovernance:
		return obj.GetRecurringGovernance(), nil
	default:
		return nil, ErrUnsupportedTransferKind
	}
}

type recurringTransferResolver VegaResolverRoot

func (r *recurringTransferResolver) StartEpoch(ctx context.Context, obj *eventspb.RecurringTransfer) (int, error) {
	return int(obj.StartEpoch), nil
}

func (r *recurringTransferResolver) EndEpoch(ctx context.Context, obj *eventspb.RecurringTransfer) (*int, error) {
	if obj.EndEpoch != nil {
		i := int(*obj.EndEpoch)
		return &i, nil
	}
	return nil, nil
}

func (r *recurringTransferResolver) DispatchStrategy(ctx context.Context, obj *eventspb.RecurringTransfer) (*DispatchStrategy, error) {
	if obj.DispatchStrategy != nil {
		return &DispatchStrategy{
			DispatchMetric:        obj.DispatchStrategy.Metric,
			DispatchMetricAssetID: obj.DispatchStrategy.AssetForMetric,
			MarketIdsInScope:      obj.DispatchStrategy.Markets,
		}, nil
	}
	return nil, nil
}

type recurringGovernanceTransferResolver VegaResolverRoot

func (r *recurringGovernanceTransferResolver) StartEpoch(ctx context.Context, obj *eventspb.RecurringGovernanceTransfer) (int, error) {
	return int(obj.StartEpoch), nil
}

func (r *recurringGovernanceTransferResolver) EndEpoch(ctx context.Context, obj *eventspb.RecurringGovernanceTransfer) (*int, error) {
	if obj.EndEpoch != nil {
		i := int(*obj.EndEpoch)
		return &i, nil
	}
	return nil, nil
}

func (r *recurringGovernanceTransferResolver) DispatchStrategy(ctx context.Context, obj *eventspb.RecurringGovernanceTransfer) (*DispatchStrategy, error) {
	if obj.DispatchStrategy != nil {
		return &DispatchStrategy{
			DispatchMetric:        obj.DispatchStrategy.Metric,
			DispatchMetricAssetID: obj.DispatchStrategy.AssetForMetric,
			MarketIdsInScope:      obj.DispatchStrategy.Markets,
		}, nil
	}
	return nil, nil
}
