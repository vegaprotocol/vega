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

package gql

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/vegatime"
	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

var ErrUnsupportedTransferKind = errors.New("unsupported transfer kind")

type transferResolver VegaResolverRoot

func (r *transferResolver) Asset(ctx context.Context, obj *eventspb.Transfer) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *transferResolver) Status(ctx context.Context, obj *eventspb.Transfer) (TransferStatus, error) {
	return convertTransferStatusFromProto(obj.Status)
}

func (r *transferResolver) Timestamp(ctx context.Context, obj *eventspb.Transfer) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

func (r *transferResolver) Kind(ctx context.Context, obj *eventspb.Transfer) (TransferKind, error) {
	switch obj.GetKind().(type) {
	case *eventspb.Transfer_OneOff:
		return obj.GetOneOff(), nil
	case *eventspb.Transfer_Recurring:
		return obj.GetRecurring(), nil
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
		metric, err := dispatchMetricFromProto(obj.DispatchStrategy.Metric)
		if err != nil {
			return nil, err
		}
		return &DispatchStrategy{
			DispatchMetric:        metric,
			DispatchMetricAssetID: obj.DispatchStrategy.AssetForMetric,
			MarketIdsInScope:      obj.DispatchStrategy.Markets,
		}, nil
	}
	return nil, nil
}

func dispatchMetricFromProto(s vega.DispatchMetric) (DispatchMetric, error) {
	switch s {
	case vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED:
		return DispatchMetricLPFeesReceived, nil
	case vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED:
		return DispatchMetricMakerFeesReceived, nil
	case vega.DispatchMetric_DISPATCH_METRIC_TAKER_FEES_PAID:
		return DispatchMetricTakerFeesPaid, nil
	case vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE:
		return DispatchMetricMarketTradingValue, nil

	default:
		return DispatchMetric(""), fmt.Errorf("failed to convert dispatch metric from Proto to GraphQL: %s", s.String())
	}
}

type oneoffTransferResolver VegaResolverRoot

func (r *oneoffTransferResolver) DeliverOn(ctx context.Context, obj *eventspb.OneOffTransfer) (*string, error) {
	if obj.DeliverOn > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.DeliverOn))
		return &t, nil
	}
	return nil, nil
}
