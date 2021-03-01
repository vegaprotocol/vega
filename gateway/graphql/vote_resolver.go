package gql

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type voteResolver VegaResolverRoot

func (r *voteResolver) Value(ctx context.Context, obj *proto.Vote) (VoteValue, error) {
	return convertVoteValueFromProto(obj.Value)
}
func (r *voteResolver) Party(ctx context.Context, obj *proto.Vote) (*proto.Party, error) {
	return &proto.Party{Id: obj.PartyId}, nil
}

func (r *voteResolver) Datetime(ctx context.Context, obj *proto.Vote) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
