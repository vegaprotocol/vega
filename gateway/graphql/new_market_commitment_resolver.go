package gql

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
)

type newMarketCommitmentResolver VegaResolverRoot

func (r *newMarketCommitmentResolver) CommitmentAmount(ctx context.Context, obj *proto.NewMarketCommitment) (string, error) {
	return obj.CommitmentAmount, nil
}
