package node

import (
	"context"
	"errors"
	"sync/atomic"

	walletapi "code.vegaprotocol.io/vega/wallet/api"

	"go.uber.org/zap"
)

var (
	ErrNoNodeConfigured       = errors.New("no node configured on round-robin selector")
	ErrNoHealthyNodeAvailable = errors.New("no healthy node available")
)

// RoundRobinSelector uses a classic round-robin algorithm to select a node.
// When requesting the next node, this is the node right behind the current one
// that is selected. When the last node is reached, it starts over the first one.
type RoundRobinSelector struct {
	log *zap.Logger

	// currentAbsoluteIndex is the index used to determine which node is the
	// next. It's converted into a relative index that points to a Node instance
	// in the `nodes` property.
	currentAbsoluteIndex uint64

	// nodes is the list of the nodes we are connected to.
	nodes []walletapi.Node
}

func (ns *RoundRobinSelector) Node(ctx context.Context) (walletapi.Node, error) {
	for i := 0; i < len(ns.nodes); i++ {
		nextAbsoluteIndex := atomic.AddUint64(&ns.currentAbsoluteIndex, 1)
		nextRelativeIndex := (int(nextAbsoluteIndex) - 1) % len(ns.nodes)
		nextNode := ns.nodes[nextRelativeIndex]
		ns.log.Info("moved to next node",
			zap.String("host", nextNode.Host()),
			zap.Int("index", nextRelativeIndex),
		)
		err := nextNode.HealthCheck(ctx)
		if err == nil {
			ns.log.Info("selected node is healthy",
				zap.String("host", nextNode.Host()),
				zap.Int("index", nextRelativeIndex),
			)
			return nextNode, nil
		}
		ns.log.Error("selected node is unhealthy",
			zap.String("host", nextNode.Host()),
			zap.Int("index", nextRelativeIndex),
		)
	}
	ns.log.Error("no healthy node available")
	return nil, ErrNoHealthyNodeAvailable
}

// Stop stops all the registered nodes. If a node raises an error during
// closing, the selector ignores it and carry on a best-effort.
func (ns *RoundRobinSelector) Stop() {
	for _, n := range ns.nodes {
		// Ignoring errors to ensure we close as many connections as possible.
		_ = n.Stop()
	}
	ns.log.Info("Stopped all the nodes")
}

func NewRoundRobinSelector(log *zap.Logger, nodes ...walletapi.Node) (*RoundRobinSelector, error) {
	if len(nodes) == 0 {
		return nil, ErrNoNodeConfigured
	}

	return &RoundRobinSelector{
		log:                  log,
		currentAbsoluteIndex: 0,
		nodes:                nodes,
	}, nil
}
