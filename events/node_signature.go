package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

// NodeSignature ...
type NodeSignature struct {
	*Base
	e types.NodeSignature
}

func NewNodeSignatureEvent(ctx context.Context, e types.NodeSignature) *NodeSignature {
	return &NodeSignature{
		Base: newBase(ctx, NodeSignatureEvent),
		e:    e,
	}
}

func (n NodeSignature) NodeSignature() types.NodeSignature {
	return n.e
}

func (n NodeSignature) Proto() types.NodeSignature {
	return n.e
}
