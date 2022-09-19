package node

import (
	"fmt"

	"go.uber.org/zap"
)

func BuildRoundRobinSelectorWithRetryingNodes(log *zap.Logger, hosts []string, retries uint64) (Selector, error) {
	nodes := make([]Node, 0, len(hosts))
	for _, host := range hosts {
		n, err := NewRetryingGRPCNode(log.Named("grpc-node"), host, retries)
		if err != nil {
			return nil, fmt.Errorf("could not initialize the node %q: %w", host, err)
		}
		nodes = append(nodes, n)
	}

	nodeSelector, err := NewRoundRobinSelector(log.Named("round-robin-selector"), nodes...)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate the round-robin node selector: %w", err)
	}

	return nodeSelector, nil
}
