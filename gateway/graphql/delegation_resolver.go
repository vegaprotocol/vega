package gql

import (
	"context"
	"strconv"

	proto "code.vegaprotocol.io/protos/vega"
)

type delegationResolver VegaResolverRoot

func (r *delegationResolver) Party(ctx context.Context, obj *proto.Delegation) (*proto.Party, error) {
	return &proto.Party{Id: obj.Party}, nil
}

func (r *delegationResolver) Node(ctx context.Context, obj *proto.Delegation) (*proto.Node, error) {
	return r.r.getNodeByID(ctx, obj.NodeId)
}

func (r *delegationResolver) Epoch(ctx context.Context, obj *proto.Delegation) (int, error) {
	seq, err := strconv.Atoi(obj.EpochSeq)
	if err != nil {
		return -1, err
	}

	return seq, nil
}
