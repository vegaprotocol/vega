package gql

import (
	"context"
	"fmt"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	proto "code.vegaprotocol.io/protos/vega"
)

type nodeResolver VegaResolverRoot

func (r *nodeResolver) Status(ctx context.Context, obj *proto.Node) (NodeStatus, error) {
	return nodeStatusFromProto(obj.Status)
}

func (r *nodeResolver) Delegations(
	ctx context.Context,
	obj *proto.Node,
	partyID *string,
	skip, first, last *int,
) ([]*proto.Delegation, error) {

	req := &protoapi.DelegationsRequest{
		NodeId:     obj.Id,
		Pagination: makePagination(skip, first, last),
	}

	if partyID != nil && *partyID != "" {
		req.Party = *partyID
	}

	resp, err := r.tradingDataClient.Delegations(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Delegations, nil
}

func nodeStatusFromProto(s proto.NodeStatus) (NodeStatus, error) {
	switch s {
	case proto.NodeStatus_NODE_STATUS_VALIDATOR:
		return NodeStatusValidator, nil
	case proto.NodeStatus_NODE_STATUS_NON_VALIDATOR:
		return NodeStatusNonValidator, nil
	default:
		return NodeStatus(""), fmt.Errorf("failed to convert NodeStatus from Proto to GraphQL: %s", s.String())
	}
}

func (r *nodeResolver) RankingScore(ctx context.Context, obj *proto.Node) (proto.RankingScore, error) {
	return *obj.RankingScore, nil
}

func (r *nodeResolver) RewardScore(ctx context.Context, obj *proto.Node) (proto.RewardScore, error) {
	return *obj.RewardScore, nil
}
