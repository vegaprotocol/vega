package gql

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
)

//type LiquidityProvisionUpdateResolver interface {
//	CreatedAt(ctx context.Context, obj *vega.LiquidityProvision) (string, error)
//	UpdatedAt(ctx context.Context, obj *vega.LiquidityProvision) (*string, error)
//
//	Version(ctx context.Context, obj *vega.LiquidityProvision) (string, error)
//}

type liquidityProvisionUpdateResolver VegaResolverRoot

func (r *liquidityProvisionUpdateResolver) CreatedAt(ctx context.Context, obj *vega.LiquidityProvision) (string, error) {
	return strconv.FormatInt(obj.CreatedAt, 10), nil
}

func (r *liquidityProvisionUpdateResolver) UpdatedAt(ctx context.Context, obj *vega.LiquidityProvision) (*string, error) {
	if obj.UpdatedAt == 0 {
		return nil, nil
	}

	updatedAt := strconv.FormatInt(obj.UpdatedAt, 10)

	return &updatedAt, nil
}

func (r *liquidityProvisionUpdateResolver) Version(ctx context.Context, obj *vega.LiquidityProvision) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}
