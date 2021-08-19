package gql

import (
	"context"
	"strconv"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
)

type rewardDetailsResolver VegaResolverRoot

func (r *rewardDetailsResolver) Details(ctx context.Context, obj *protoapi.GetRewardDetailsResponse) ([]*RewardPerAssetDetails, error) {
	// Create the empty slice
	rpads := make([]*RewardPerAssetDetails, 0, len(obj.RewardDetails))

	// Now copy across the information in the new structure.
	for _, rpad := range obj.RewardDetails {
		rpa := &RewardPerAssetDetails{
			AssetID:     rpad.AssetId,
			TotalAmount: rpad.TotalForAsset,
			Rewards:     make([]*Reward, 0),
		}
		for _, r := range rpad.Details {
			reward := Reward{
				AssetID:           r.AssetId,
				PartyID:           r.PartyId,
				Epoch:             int(r.Epoch),
				Amount:            r.Amount,
				PercentageOfTotal: r.PercentageOfTotal,
				ReceivedAt:        strconv.FormatInt(r.ReceivedAt, 10),
			}
			rpa.Rewards = append(rpa.Rewards, &reward)
		}
		rpads = append(rpads, rpa)
	}
	return rpads, nil
}
