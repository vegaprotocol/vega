package gql

import (
	"context"
	"strconv"

	proto "code.vegaprotocol.io/protos/vega"
)

type delegationResolver VegaResolverRoot

func (r *delegationResolver) PartyID(ctx context.Context, obj *proto.Delegation) (string, error) {
	return obj.Party, nil
}

func (r *delegationResolver) Node(ctx context.Context, obj *proto.Delegation) (string, error) {
	return obj.NodeId, nil
}

func (r *delegationResolver) Epoch(ctx context.Context, obj *proto.Delegation) (int, error) {
	seq, err := strconv.Atoi(obj.EpochSeq)
	if err != nil {
		return -1, err
	}

	return seq, nil
}
