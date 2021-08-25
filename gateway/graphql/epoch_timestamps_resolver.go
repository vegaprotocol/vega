package gql

import (
	"context"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
)

type epochTimestampsResolver VegaResolverRoot

func (r *epochTimestampsResolver) Start(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	t := time.Unix(obj.StartTime, 0).Format(time.RFC3339)

	return &t, nil
}

func (r *epochTimestampsResolver) End(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	t := time.Unix(obj.EndTime, 0).Format(time.RFC3339)

	return &t, nil
}
