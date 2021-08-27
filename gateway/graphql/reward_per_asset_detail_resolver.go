package gql

import (
	"context"

	"code.vegaprotocol.io/data-node/vegatime"
	"code.vegaprotocol.io/protos/vega"
)

type rewardPerAssetDetailResolver VegaResolverRoot

func (r *rewardPerAssetDetailResolver) Asset(ctx context.Context, obj *vega.RewardPerAssetDetail) (*vega.Asset, error) {
	asset, err := r.r.getAssetByID(ctx, obj.Asset)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

func (r *rewardPerAssetDetailResolver) Rewards(ctx context.Context, obj *vega.RewardPerAssetDetail) ([]*Reward, error) {
	rewards := make([]*Reward, 0, len(obj.Details))

	// Now copy across the information in the new structure.
	for _, rd := range obj.Details {
		reward := Reward{
			AssetID:           rd.AssetId,
			PartyID:           rd.PartyId,
			Epoch:             int(rd.Epoch),
			Amount:            rd.Amount,
			PercentageOfTotal: rd.PercentageOfTotal,
			ReceivedAt:        vegatime.Format(vegatime.UnixNano(rd.ReceivedAt)),
		}

		rewards = append(rewards, &reward)
	}

	return rewards, nil
}

func (r *rewardPerAssetDetailResolver) TotalAmount(ctx context.Context, obj *vega.RewardPerAssetDetail) (string, error) {
	return obj.TotalForAsset, nil
}
