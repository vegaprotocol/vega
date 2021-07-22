package gql

import (
	"context"
	"strconv"

	proto "code.vegaprotocol.io/data-node/proto/vega"
)

type newMarketCommitmentResolver VegaResolverRoot

func (r *newMarketCommitmentResolver) CommitmentAmount(ctx context.Context, obj *proto.NewMarketCommitment) (string, error) {
	return strconv.FormatUint(obj.CommitmentAmount, 10), nil
}
