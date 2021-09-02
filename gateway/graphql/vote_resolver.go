package gql

import (
	"context"

	"code.vegaprotocol.io/data-node/vegatime"
	proto "code.vegaprotocol.io/protos/vega"
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
	return obj.TotalGovernanceTokenBalance, nil
}

func (r *voteResolver) GovernanceTokenWeight(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalGovernanceTokenWeight, nil
}
