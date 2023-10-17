package gql

import (
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"golang.org/x/net/context"
)

type entitiesResolver VegaResolverRoot

func (r *entitiesResolver) LiquidityProvisions(ctx context.Context, obj *v2.ListEntitiesResponse) ([]*v2.LiquidityProvision, error) {
	lps := make([]*v2.LiquidityProvision, len(obj.LiquidityProvisions))
	for i, lp := range obj.LiquidityProvisions {
		lps[i] = &v2.LiquidityProvision{
			Id:               lp.Id,
			PartyId:          lp.PartyId,
			CreatedAt:        lp.CreatedAt,
			UpdatedAt:        lp.UpdatedAt,
			MarketId:         lp.MarketId,
			CommitmentAmount: lp.CommitmentAmount,
			Fee:              lp.Fee,
			Sells:            lp.Sells,
			Buys:             lp.Buys,
			Version:          lp.Version,
			Status:           lp.Status,
			Reference:        lp.Reference,
		}
	}
	return lps, nil
}
