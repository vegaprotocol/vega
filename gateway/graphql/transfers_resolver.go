package gql

import (
	"context"
	"errors"

	"code.vegaprotocol.io/data-node/vegatime"
	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

var ErrUnsupportedTransferKind = errors.New("unsupported transfer kind")

type transferResolver VegaResolverRoot

func (r *transferResolver) MarketID(_ context.Context, obj *eventspb.Transfer) (string, error) {
	return obj.Market, nil
}

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
		var i int = int(obj.EndEpoch.Value)
		return &i, nil
	}
	return nil, nil
}

type oneoffTransferResolver VegaResolverRoot

func (r *oneoffTransferResolver) DeliverOn(ctx context.Context, obj *eventspb.OneOffTransfer) (*string, error) {
	if obj.DeliverOn > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.DeliverOn))
		return &t, nil
	}
	return nil, nil
}
