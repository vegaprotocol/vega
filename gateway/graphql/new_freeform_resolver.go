package gql

import (
	"context"

	"code.vegaprotocol.io/protos/vega"
)

type newFreeformResolver VegaResolverRoot

func (r *newFreeformResolver) DoNotUse(_ context.Context, _ *vega.NewFreeform) (*bool, error) {
	shouldUse := false
	return &shouldUse, nil
}
