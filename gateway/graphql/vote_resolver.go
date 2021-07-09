package gql

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type voteResolver VegaResolverRoot

func (r *voteResolver) Value(_ context.Context, obj *proto.Vote) (VoteValue, error) {
	return convertVoteValueFromProto(obj.Value)
}
func (r *voteResolver) Party(_ context.Context, obj *proto.Vote) (*proto.Party, error) {
	return &proto.Party{Id: obj.PartyId}, nil
}

func (r *voteResolver) Datetime(_ context.Context, obj *proto.Vote) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

func (r *voteResolver) GovernanceTokenBalance(_ context.Context, obj *proto.Vote) (string, error) {
	return strconv.FormatUint(obj.TotalGovernanceTokenBalance, 10), nil
}

func (r *voteResolver) GovernanceTokenWeight(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalGovernanceTokenWeight, nil
}
