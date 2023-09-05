package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type referralSetRefereeResolver VegaResolverRoot

func (r *referralSetRefereeResolver) AtEpoch(ctx context.Context, obj *v2.ReferralSetReferee) (int, error) {
	if obj == nil {
		return 0, nil
	}

	return int(obj.AtEpoch), nil
}
