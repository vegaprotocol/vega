package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type updateNetworkParameterResolver VegaResolverRoot

func (r *updateNetworkParameterResolver) NetworkParameter(ctx context.Context, obj *types.UpdateNetworkParameter) (*types.NetworkParameter, error) {
	return obj.Changes, nil
}
