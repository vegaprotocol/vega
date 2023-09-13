package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type teamResolver VegaResolverRoot

func (t teamResolver) CreatedAtEpoch(_ context.Context, obj *v2.Team) (int, error) {
	return int(obj.CreatedAtEpoch), nil
}

type teamRefereeResolver VegaResolverRoot

func (t teamRefereeResolver) JoinedAtEpoch(_ context.Context, obj *v2.TeamReferee) (int, error) {
	return int(obj.JoinedAtEpoch), nil
}

type teamRefereeHistoryResolver VegaResolverRoot

func (t teamRefereeHistoryResolver) JoinedAtEpoch(_ context.Context, obj *v2.TeamRefereeHistory) (int, error) {
	return int(obj.JoinedAtEpoch), nil
}
