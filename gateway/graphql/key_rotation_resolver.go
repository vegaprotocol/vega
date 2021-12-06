package gql

import (
	"context"
	"strconv"

	v12 "code.vegaprotocol.io/protos/data-node/api/v1"
)

type keyRotationResolver VegaResolverRoot

func (r *keyRotationResolver) BlockHeight(ctx context.Context, obj *v12.KeyRotation) (string, error) {
	return strconv.FormatUint(obj.BlockHeight, 10), nil
}
