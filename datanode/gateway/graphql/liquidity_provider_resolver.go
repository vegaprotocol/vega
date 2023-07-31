package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type liquidityProviderResolver VegaResolverRoot

func (r *liquidityProviderResolver) FeeShare(_ context.Context, obj *v2.LiquidityProvider) (*LiquidityProviderFeeShare, error) {
	return &LiquidityProviderFeeShare{
		Party:                 &vegapb.Party{Id: obj.PartyId},
		EquityLikeShare:       obj.FeeShare.EquityLikeShare,
		AverageEntryValuation: obj.FeeShare.AverageEntryValuation,
		AverageScore:          obj.FeeShare.AverageScore,
		VirtualStake:          obj.FeeShare.VirtualStake,
	}, nil
}
