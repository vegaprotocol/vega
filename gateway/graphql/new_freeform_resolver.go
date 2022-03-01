package gql

import (
	"context"

	types "code.vegaprotocol.io/protos/vega"
)

type newFreeformResolver VegaResolverRoot

func (r newFreeformResolver) URL(ctx context.Context, obj *types.NewFreeform) (string, error) {
	return obj.Changes.Url, nil
}

func (r newFreeformResolver) Description(ctx context.Context, obj *types.NewFreeform) (string, error) {
	return obj.Changes.Description, nil
}

func (r newFreeformResolver) Hash(ctx context.Context, obj *types.NewFreeform) (string, error) {
	return obj.Changes.Hash, nil
}
