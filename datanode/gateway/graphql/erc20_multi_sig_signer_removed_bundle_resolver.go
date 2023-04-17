package gql

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type erc20MultiSigSignerRemovedBundleResolver VegaResolverRoot

func (e erc20MultiSigSignerRemovedBundleResolver) Timestamp(ctx context.Context, obj *v2.ERC20MultiSigSignerRemovedBundle) (string, error) {
	return fmt.Sprint(obj.Timestamp), nil
}
