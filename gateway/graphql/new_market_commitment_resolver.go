package gql

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/proto"
)

type newMarketCommitmentResolver VegaResolverRoot

func (r *newMarketCommitmentResolver) CommitmentAmount(ctx context.Context, obj *proto.NewMarketCommitment) (string, error) {
	return strconv.FormatUint(obj.CommitmentAmount, 10), nil
}
