package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vega "code.vegaprotocol.io/vega/protos/vega"
)

type myAccountEdgeResolver VegaResolverRoot

func (r *myAccountEdgeResolver) Node(_ context.Context, edge *v2.AccountEdge) (*vega.Account, error) {
	return edge.Account, nil
}
