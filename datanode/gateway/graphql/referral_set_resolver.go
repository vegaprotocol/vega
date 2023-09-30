package gql

import (
	"context"
	"errors"
	"math"

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

type referralSetStatsResolver VegaResolverRoot

func (r *referralSetStatsResolver) AtEpoch(_ context.Context, obj *v2.ReferralSetStats) (int, error) {
	if obj.AtEpoch > math.MaxInt {
		return 0, errors.New("at_epoch is too large")
	}

	return int(obj.AtEpoch), nil
}
