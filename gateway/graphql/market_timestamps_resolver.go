package gql

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type marketTimestampsResolver VegaResolverRoot

func (r *marketTimestampsResolver) Proposed(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Proposed == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Proposed))
	return &value, nil
}

func (r *marketTimestampsResolver) Pending(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Pending == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Pending))
	return &value, nil
}

func (r *marketTimestampsResolver) Open(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Open == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Open))
	return &value, nil
}

func (r *marketTimestampsResolver) Close(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Close == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Close))
	return &value, nil
}
