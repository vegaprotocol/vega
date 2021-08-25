package gql

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
)

type nodeDataResolver VegaResolverRoot

func (r *nodeDataResolver) TotalNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.TotalNodes), nil
}

func (r *nodeDataResolver) InactiveNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.InactiveNodes), nil
}

func (r *nodeDataResolver) ValidatingNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.ValidatingNodes), nil
}

func (r *nodeDataResolver) Uptime(ctx context.Context, obj *proto.NodeData) (float64, error) {
	return float64(obj.Uptime), nil
}
