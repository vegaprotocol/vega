package gql

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type marketTimestampsResolver VegaResolverRoot

func (r *marketTimestampsResolver) Proposed(ctx context.Context, obj *proto.MarketTimestamps) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Proposed)), nil
}

func (r *marketTimestampsResolver) Pending(ctx context.Context, obj *proto.MarketTimestamps) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Pending)), nil
}

func (r *marketTimestampsResolver) Open(ctx context.Context, obj *proto.MarketTimestamps) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Open)), nil
}

func (r *marketTimestampsResolver) Close(ctx context.Context, obj *proto.MarketTimestamps) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Close)), nil
}
