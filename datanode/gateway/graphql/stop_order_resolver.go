package gql

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type stopOrderResolver VegaResolverRoot

func (s stopOrderResolver) ID(_ context.Context, obj *eventspb.StopOrderEvent) (string, error) {
	if obj == nil || obj.StopOrder == nil {
		return "", ErrInvalidStopOrder
	}
	return obj.StopOrder.Id, nil
}

func (s stopOrderResolver) OcoLinkID(_ context.Context, obj *eventspb.StopOrderEvent) (*string, error) {
	if obj == nil || obj.StopOrder == nil {
		return nil, ErrInvalidStopOrder
	}
	return obj.StopOrder.OcoLinkId, nil
}

func (s stopOrderResolver) ExpiresAt(_ context.Context, obj *eventspb.StopOrderEvent) (*int64, error) {
	if obj == nil || obj.StopOrder == nil {
		return nil, ErrInvalidStopOrder
	}
	return obj.StopOrder.ExpiresAt, nil
}

func (s stopOrderResolver) ExpiryStrategy(_ context.Context, obj *eventspb.StopOrderEvent) (*vega.StopOrder_ExpiryStrategy, error) {
	if obj == nil || obj.StopOrder == nil {
		return nil, ErrInvalidStopOrder
	}
	return obj.StopOrder.ExpiryStrategy, nil
}

func (s stopOrderResolver) TriggerDirection(_ context.Context, obj *eventspb.StopOrderEvent) (vega.StopOrder_TriggerDirection, error) {
	if obj == nil || obj.StopOrder == nil {
		return vega.StopOrder_TRIGGER_DIRECTION_UNSPECIFIED, ErrInvalidStopOrder
	}
	return obj.StopOrder.TriggerDirection, nil
}

func (s stopOrderResolver) Status(_ context.Context, obj *eventspb.StopOrderEvent) (vega.StopOrder_Status, error) {
	if obj == nil || obj.StopOrder == nil {
		return vega.StopOrder_STATUS_UNSPECIFIED, ErrInvalidStopOrder
	}
	return obj.StopOrder.Status, nil
}

func (s stopOrderResolver) CreatedAt(_ context.Context, obj *eventspb.StopOrderEvent) (int64, error) {
	if obj == nil || obj.StopOrder == nil {
		return 0, ErrInvalidStopOrder
	}
	return obj.StopOrder.CreatedAt, nil
}

func (s stopOrderResolver) UpdatedAt(_ context.Context, obj *eventspb.StopOrderEvent) (*int64, error) {
	if obj == nil || obj.StopOrder == nil {
		return nil, ErrInvalidStopOrder
	}
	return obj.StopOrder.UpdatedAt, nil
}

func (s stopOrderResolver) PartyID(_ context.Context, obj *eventspb.StopOrderEvent) (string, error) {
	if obj == nil || obj.StopOrder == nil {
		return "", ErrInvalidStopOrder
	}
	return obj.StopOrder.PartyId, nil
}

func (s stopOrderResolver) MarketID(_ context.Context, obj *eventspb.StopOrderEvent) (string, error) {
	if obj == nil || obj.StopOrder == nil {
		return "", ErrInvalidStopOrder
	}
	return obj.StopOrder.MarketId, nil
}

func (s stopOrderResolver) Trigger(_ context.Context, obj *eventspb.StopOrderEvent) (StopOrderTrigger, error) {
	if obj == nil || obj.StopOrder == nil {
		return nil, ErrInvalidStopOrder
	}
	switch t := obj.StopOrder.Trigger.(type) {
	case *vega.StopOrder_Price:
		return StopOrderPrice{
			Price: t.Price,
		}, nil
	case *vega.StopOrder_TrailingPercentOffset:
		return StopOrderTrailingPercentOffset{
			TrailingPercentOffset: t.TrailingPercentOffset,
		}, nil
	default:
		return nil, fmt.Errorf("unknown trigger type: %T", t)
	}
}

func (s stopOrderResolver) Order(ctx context.Context, obj *eventspb.StopOrderEvent) (*vega.Order, error) {
	// no order triggeerd yet
	if len(obj.StopOrder.OrderId) <= 0 {
		return nil, nil
	}

	return s.r.getOrderByID(ctx, obj.StopOrder.OrderId, nil)
}

type stopOrderFilterResolver VegaResolverRoot

func (s stopOrderFilterResolver) Parties(ctx context.Context, obj *v2.StopOrderFilter, data []string) error {
	if obj == nil {
		obj = &v2.StopOrderFilter{}
	}
	obj.PartyIds = data
	return nil
}

func (s stopOrderFilterResolver) Markets(ctx context.Context, obj *v2.StopOrderFilter, data []string) error {
	if obj == nil {
		obj = &v2.StopOrderFilter{}
	}
	obj.MarketIds = data
	return nil
}

func (s stopOrderFilterResolver) Status(ctx context.Context, obj *v2.StopOrderFilter, data []vega.StopOrder_Status) error {
	if obj == nil {
		obj = &v2.StopOrderFilter{}
	}
	obj.Statuses = data
	return nil
}

func (s stopOrderFilterResolver) ExpiryStrategy(ctx context.Context, obj *v2.StopOrderFilter, data []vega.StopOrder_ExpiryStrategy) error {
	if obj == nil {
		obj = &v2.StopOrderFilter{}
	}
	obj.ExpiryStrategies = data
	return nil
}
