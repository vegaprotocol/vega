package gql

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	partyAmountResolver              VegaResolverRoot
	referralSetFeeStatsResolver      VegaResolverRoot
	referrerRewardsGeneratedResolver VegaResolverRoot
)

func (r *partyAmountResolver) PartyID(_ context.Context, obj *eventspb.PartyAmount) (string, error) {
	return obj.Party, nil
}

func (r *referralSetFeeStatsResolver) MarketID(ctx context.Context, obj *eventspb.FeeStats) (string, error) {
	return obj.Market, nil
}

func (r *referralSetFeeStatsResolver) AssetID(ctx context.Context, obj *eventspb.FeeStats) (string, error) {
	return obj.Asset, nil
}

func (r *referralSetFeeStatsResolver) Epoch(ctx context.Context, obj *eventspb.FeeStats) (int, error) {
	return int(obj.EpochSeq), nil
}

func (r *referrerRewardsGeneratedResolver) ReferrerID(ctx context.Context, obj *eventspb.ReferrerRewardsGenerated) (string, error) {
	return obj.Referrer, nil
}
