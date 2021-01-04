package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type myFutureProductResolver VegaResolverRoot

func (r *myFutureProductResolver) Asset(ctx context.Context, obj *types.FutureProduct) (*Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}
