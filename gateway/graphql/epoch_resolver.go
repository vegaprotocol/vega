package gql

import (
	"context"
	"strconv"

	proto "code.vegaprotocol.io/protos/vega"
)

type epochResolver VegaResolverRoot

func (r *epochResolver) ID(ctx context.Context, obj *proto.Epoch) (string, error) {
	id := strconv.FormatUint(obj.Seq, 10)

	return id, nil
}
