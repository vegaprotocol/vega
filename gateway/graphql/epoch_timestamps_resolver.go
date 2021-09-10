package gql

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/vegatime"
	proto "code.vegaprotocol.io/protos/vega"
)

type epochTimestampsResolver VegaResolverRoot

func (r *epochTimestampsResolver) Start(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var t string
	if obj.StartTime > 0 {
		t = vegatime.UnixNano(obj.StartTime).Format(time.RFC3339)
	}
	return &t, nil
}

func (r *epochTimestampsResolver) End(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var t string
	if obj.EndTime > 0 {
		t = vegatime.UnixNano(obj.EndTime).Format(time.RFC3339)
	}
	return &t, nil
}

func (r *epochTimestampsResolver) Expiry(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var t string
	if obj.ExpiryTime > 0 {
		t = vegatime.UnixNano(obj.ExpiryTime).Format(time.RFC3339)
	}
	return &t, nil
}
