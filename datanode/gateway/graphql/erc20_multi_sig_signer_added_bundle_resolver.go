package gql

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type erc20MultiSigSignerAddedBundleResolver VegaResolverRoot

func (e erc20MultiSigSignerAddedBundleResolver) Timestamp(ctx context.Context, obj *v2.ERC20MultiSigSignerAddedBundle) (string, error) {
	return fmt.Sprint(obj.Timestamp), nil
}
