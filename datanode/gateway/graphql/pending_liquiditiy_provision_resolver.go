package gql

import (
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
	"golang.org/x/net/context"
)

type pendingLiquidityProvisionResolver VegaResolverRoot

func (r *pendingLiquidityProvisionResolver) Party(ctx context.Context, obj *vega.LiquidityProvision) (*vega.Party, error) {
	return &vega.Party{Id: obj.PartyId}, nil
}

func (r *pendingLiquidityProvisionResolver) Market(ctx context.Context, obj *vega.LiquidityProvision) (*vega.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *pendingLiquidityProvisionResolver) Version(ctx context.Context, obj *vega.LiquidityProvision) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}
