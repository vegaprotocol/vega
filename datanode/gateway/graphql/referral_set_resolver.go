package gql

import (
	"context"

	"code.vegaprotocol.io/vega/libs/ptr"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type referralSetRefereeResolver VegaResolverRoot

func (r *referralSetRefereeResolver) AtEpoch(ctx context.Context, obj *v2.ReferralSetReferee) (int, error) {
	if obj == nil {
		return 0, nil
	}

	return int(obj.AtEpoch), nil
}

func (r *referralSetRefereeResolver) RefereeID(ctx context.Context, obj *v2.ReferralSetReferee) (string, error) {
	return obj.Referee, nil
}

type referralSetResolver VegaResolverRoot

func (r *referralSetResolver) Stats(ctx context.Context, obj *v2.ReferralSet, epoch *int, referee *string) (*v2.ReferralSetStats, error) {
	var atEpoch *uint64
	if epoch != nil {
		atEpoch = ptr.From(uint64(*epoch))
	}

	req := v2.GetReferralSetStatsRequest{
		ReferralSetId: obj.Id,
		AtEpoch:       atEpoch,
		Referee:       referee,
	}

	res, err := r.tradingDataClientV2.GetReferralSetStats(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.Stats, nil
}

type referralSetStatsResolver VegaResolverRoot

func (r *referralSetStatsResolver) AtEpoch(_ context.Context, obj *v2.ReferralSetStats) (*int, error) {
	if obj == nil {
		return nil, nil
	}

	return ptr.From(int(obj.AtEpoch)), nil
}
