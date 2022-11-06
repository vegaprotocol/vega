package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
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

	// currentIndex is the index used to determine which node is returned.
	currentIndex *atomic.Int64

	// nodes is the list of the nodes we are connected to.
	nodes []Node
}

// Node returns the next node in line among the healthiest nodes.
//
// Algorithm:
//  1. It gets the statistics of the nodes configured
//  2. It filters out the nodes that returns data different from the majority,
//     and label those left as the "healthiest" nodes.
//  3. It tries to resolve the next node in line, based on the previous selection
//     and availability of the node. If the next node that should have selected
//     is not healthy, it skips the node. It applies this logic until it ends up
//     on a healthy node.
//
// Warning:
// We look for the network information that are the most commonly shared among
// the nodes, because, in decentralized system, the most commonly shared data
// represents the truth. While true from the entire network point of view, on a
// limited subset of nodes, this might not be true. If most of the nodes
// set up in the configuration are late, or misbehaving, the algorithm will
// fail to identify the truly healthy ones. That's the major reason to favour
// highly trusted and stable nodes.
func (ns *RoundRobinSelector) Node(ctx context.Context, reporterFn SelectionReporter) (Node, error) {
	healthiestNodesIndexes, err := ns.retrieveHealthiestNodes(ctx, reporterFn)
	if err != nil {
		ns.log.Error("no healthy node available")
		return nil, err
	}

	reporterFn(InfoEvent, "Starting round-robin selection of the node...")

	lowestHealthyIndex := healthiestNodesIndexes[0]
	highestHealthyIndex := healthiestNodesIndexes[len(healthiestNodesIndexes)-1]

	if lowestHealthyIndex == highestHealthyIndex {
		// We have a single healthy node, so no other choice than using it.
		return ns.selectNode(lowestHealthyIndex, reporterFn), nil
	}

	currentIndex := int(ns.currentIndex.Load())

	if currentIndex < lowestHealthyIndex || currentIndex >= highestHealthyIndex {
		// If the current index is outside the boundaries of the healthy indexes,
		// or already equal to the highest index, we get back to the first healthy
		// index.
		return ns.selectNode(lowestHealthyIndex, reporterFn), nil
	}

	selectedIndex := lowestHealthyIndex
	for _, healthyIndex := range healthiestNodesIndexes {
		if currentIndex < healthyIndex {
			// As soon as the current index is lower thant the healthy index, it
			// means we found the next healthy node to use.
			selectedIndex = healthyIndex
			break
		}
	}

	selectedNode := ns.selectNode(selectedIndex, reporterFn)

	return selectedNode, nil
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

func (ns *RoundRobinSelector) selectNode(selectedIndex int, reporterFn SelectionReporter) Node {
	ns.currentIndex.Store(int64(selectedIndex))
	selectedNode := ns.nodes[ns.currentIndex.Load()]

	reporterFn(SuccessEvent, fmt.Sprintf("The node %q has been selected", selectedNode.Host()))
	ns.log.Info("a node has been selected",
		zap.String("host", selectedNode.Host()),
		zap.Int("index", selectedIndex),
	)

	return selectedNode
}

func (ns *RoundRobinSelector) retrieveHealthiestNodes(ctx context.Context, reporterFn SelectionReporter) ([]int, error) {
	ns.log.Info("start evaluating nodes health based on each others state")

	nodeStatsHashes, err := ns.collectNodesInformation(ctx, reporterFn)
	if err != nil {
		return nil, err
	}

	nodesGroupedByHash := ns.groupNodeIndexesByHash(nodeStatsHashes)

	hashCount := len(nodesGroupedByHash)

	reporterFn(InfoEvent, "Looking for healthiest nodes...")

	rankedHashes := ns.rankHashes(hashCount, nodesGroupedByHash)

	// We return the nodes indexes that generate the same hash the most often.
	// Since the slice is sorted for the lowest to the highest occurrences,
	// the last element is the highest.
	selectedHash := rankedHashes[hashCount-1]

	healthiestNodesIndexes := selectedHash.nodesIndexes

	healthyNodesCount := len(healthiestNodesIndexes)
	if healthyNodesCount > 1 {
		reporterFn(SuccessEvent, fmt.Sprintf("%d healthy nodes found", healthyNodesCount))
	} else {
		reporterFn(SuccessEvent, "1 healthy node found")
	}
	ns.log.Info("healthy nodes found", zap.Any("node-indexes", healthiestNodesIndexes))

	return healthiestNodesIndexes, nil
}

func (ns *RoundRobinSelector) rankHashes(hashCount int, nodesGroupedByHash map[string]nodesByHash) []nodesByHash {
	rankedHashes := make([]nodesByHash, 0, hashCount)
	for _, groupedNodes := range nodesGroupedByHash {
		rankedHashes = append(rankedHashes, groupedNodes)
	}

	sort.Slice(rankedHashes, func(i, j int) bool {
		if len(rankedHashes[i].nodesIndexes) == len(rankedHashes[j].nodesIndexes) {
			// if we have the same number of nodes indexes, we select the ones that
			// have the most recent block height, as we think it's the most
			// sensible thing to do.
			// However, if they also have the same block height, nothing can be
			// done to really figure out which nodes are the healthiest one, so
			// we just ensure a deterministic sorting.
			// This can be wrong, but at least it's consistently wrong.
			if rankedHashes[i].blockHeight == rankedHashes[j].blockHeight {
				return rankedHashes[i].hash < rankedHashes[j].hash
			}
			return rankedHashes[i].blockHeight < rankedHashes[j].blockHeight
		}
		return len(rankedHashes[i].nodesIndexes) < len(rankedHashes[j].nodesIndexes)
	})

	return rankedHashes
}

func (ns *RoundRobinSelector) groupNodeIndexesByHash(nodeStatsHashes []nodeHash) map[string]nodesByHash {
	nodesGroupedByHash := map[string]nodesByHash{}
	for _, statsHash := range nodeStatsHashes {
		sh, hashAlreadyTracked := nodesGroupedByHash[statsHash.hash]
		if !hashAlreadyTracked {
			nodesGroupedByHash[statsHash.hash] = nodesByHash{
				hash:         statsHash.hash,
				blockHeight:  statsHash.blockHeight,
				nodesIndexes: []int{statsHash.index},
			}
			continue
		}

		sh.nodesIndexes = append(sh.nodesIndexes, statsHash.index)
		nodesGroupedByHash[statsHash.hash] = sh
	}
	return nodesGroupedByHash
}

func (ns *RoundRobinSelector) collectNodesInformation(ctx context.Context, reporterFn SelectionReporter) ([]nodeHash, error) {
	reporterFn(InfoEvent, "Collecting nodes information to evaluate their health...")

	nodesCount := len(ns.nodes)

	wg := sync.WaitGroup{}
	wg.Add(nodesCount)

	nodeHashes := make([]*nodeHash, nodesCount)
	for nodeIndex, node := range ns.nodes {
		_index := nodeIndex
		_node := node
		go func() {
			defer wg.Done()

			hash, blockHeight := ns.queryNodeInformation(ctx, _node, reporterFn)
			if hash == "" {
				return
			}

			nodeHashes[_index] = &nodeHash{
				hash:        hash,
				blockHeight: blockHeight,
				index:       _index,
			}
		}()
	}

	wg.Wait()

	filteredNodeHashes := []nodeHash{}
	for _, nodeHash := range nodeHashes {
		if nodeHash != nil {
			filteredNodeHashes = append(filteredNodeHashes, *nodeHash)
		}
	}

	respondingNodeCount := len(filteredNodeHashes)

	if respondingNodeCount == 0 {
		ns.log.Error("No healthy node available")
		return nil, ErrNoHealthyNodeAvailable
	}

	if respondingNodeCount > 1 {
		reporterFn(SuccessEvent, fmt.Sprintf("Successfully collected information on %d nodes", respondingNodeCount))
	} else {
		reporterFn(SuccessEvent, "Successfully collected information on 1 node")
	}

	return filteredNodeHashes, nil
}

func (ns *RoundRobinSelector) queryNodeInformation(ctx context.Context, node Node, reporterFn SelectionReporter) (string, uint64) {
	stats, err := node.Statistics(ctx)
	if err != nil {
		reporterFn(WarningEvent, fmt.Sprintf("Could not collect information from the node %q, skipping...", node.Host()))
		ns.log.Warn("Could not collect statistics for the node, skipping", zap.Error(err), zap.String("host", node.Host()))
		return "", 0
	}

	marshaledStats, err := json.Marshal(stats)
	if err != nil {
		// It's very unlikely to happen.
		reporterFn(ErrorEvent, fmt.Sprintf("[internal error] Could not prepare the collected information from the node %q for the health check", node.Host()))
		ns.log.Error("Could not marshal statistics to JSON, skipping", zap.Error(err), zap.String("host", node.Host()))
		return "", 0
	}

	ns.log.Info("The node is responding and staged for the health check", zap.String("host", node.Host()))

	return vgcrypto.HashToHex(marshaledStats), stats.BlockHeight
}

func NewRoundRobinSelector(log *zap.Logger, nodes ...Node) (*RoundRobinSelector, error) {
	if len(nodes) == 0 {
		return nil, ErrNoNodeConfigured
	}

	currentIndex := &atomic.Int64{}
	currentIndex.Store(-1)
	return &RoundRobinSelector{
		log:          log,
		currentIndex: currentIndex,
		nodes:        nodes,
	}, nil
}

type nodeHash struct {
	hash        string
	blockHeight uint64
	index       int
}

type nodesByHash struct {
	hash         string
	blockHeight  uint64
	nodesIndexes []int
}
