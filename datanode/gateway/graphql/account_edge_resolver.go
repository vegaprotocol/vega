package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type myAccountEdgeResolver VegaResolverRoot

func (r *myAccountEdgeResolver) Node(_ context.Context, edge *v2.AccountEdge) (*v2.AccountBalance, error) {
	return edge.Account, nil
}
