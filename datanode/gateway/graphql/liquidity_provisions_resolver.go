package gql

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/data-node/vegatime"
	types "code.vegaprotocol.io/protos/vega"
)

// LiquidityProvision resolver

type myLiquidityProvisionResolver VegaResolverRoot

func (r *myLiquidityProvisionResolver) Version(_ context.Context, obj *types.LiquidityProvision) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myLiquidityProvisionResolver) Party(_ context.Context, obj *types.LiquidityProvision) (*types.Party, error) {
	return &types.Party{Id: obj.PartyId}, nil
}

func (r *myLiquidityProvisionResolver) CreatedAt(ctx context.Context, obj *types.LiquidityProvision) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}

func (r *myLiquidityProvisionResolver) UpdatedAt(ctx context.Context, obj *types.LiquidityProvision) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myLiquidityProvisionResolver) Market(ctx context.Context, obj *types.LiquidityProvision) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myLiquidityProvisionResolver) CommitmentAmount(ctx context.Context, obj *types.LiquidityProvision) (string, error) {
	return obj.CommitmentAmount, nil
}

func (r *myLiquidityProvisionResolver) Status(ctx context.Context, obj *types.LiquidityProvision) (LiquidityProvisionStatus, error) {
	return convertLiquidityProvisionStatusFromProto(obj.Status)
}
