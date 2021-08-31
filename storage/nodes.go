package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	pb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

type node struct {
	n pb.Node

	delegationsPerEpochPerParty map[string]map[string]pb.Delegation
}

type Node struct {
	Config

	nodes map[string]node
	mut   sync.RWMutex

	log *logging.Logger
}

func NewNode(log *logging.Logger, c Config) *Node {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &Node{
		nodes:  map[string]node{},
		log:    log,
		Config: c,
	}
}

// ReloadConf update the internal conf of the market
func (ns *Node) ReloadConf(cfg Config) {
	ns.log.Info("reloading configuration")
	if ns.log.GetLevel() != cfg.Level.Get() {
		ns.log.Info("updating log level",
			logging.String("old", ns.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		ns.log.SetLevel(cfg.Level.Get())
	}

	ns.Config = cfg
}

func (ns *Node) AddNode(n pb.Node) {
	ns.mut.Lock()
	defer ns.mut.Unlock()

	ns.nodes[n.GetId()] = node{
		n:                           n,
		delegationsPerEpochPerParty: map[string]map[string]pb.Delegation{},
	}
}

func (ns *Node) AddDelegation(de pb.Delegation) {
	ns.mut.Lock()
	defer ns.mut.Unlock()

	node, ok := ns.nodes[de.GetNodeId()]
	if !ok {
		ns.log.Error("Received delegation balance event for non existing node", logging.String("node_id", de.GetNodeId()))
		return
	}

	if _, ok := node.delegationsPerEpochPerParty[de.GetEpochSeq()]; !ok {
		node.delegationsPerEpochPerParty[de.GetEpochSeq()] = map[string]pb.Delegation{}
	}

	node.delegationsPerEpochPerParty[de.GetEpochSeq()][de.GetParty()] = de
}

func (ns *Node) GetByID(id string) (*pb.Node, error) {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	node, ok := ns.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}

	return ns.nodeProtoFromInternal(node), nil
}

func (ns *Node) GetAll() []*pb.Node {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	nodes := make([]*pb.Node, 0, len(ns.nodes))
	for _, n := range ns.nodes {
		nodes = append(nodes, ns.nodeProtoFromInternal(n))
	}

	return nodes
}

func (ns *Node) GetAllIDs() []string {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	ids := make([]string, 0, len(ns.nodes))
	for _, n := range ns.nodes {
		ids = append(ids, n.n.GetId())
	}

	return ids
}

func (ns *Node) GetTotalNodesNumber() int {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	return len(ns.nodes)
}

// GetValidatingNodesNumber - for now this is the same as total nodes
func (ns *Node) GetValidatingNodesNumber() int {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	return len(ns.nodes)
}

// GetStakedTotal returns total stake across all nodes per epoch.
// Returns 0 if epoch not exists.
func (ns *Node) GetStakedTotal(epochID string) string {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	stakedTotal := num.NewUint(0)

	for _, n := range ns.nodes {
		dPerParty, ok := n.delegationsPerEpochPerParty[epochID]
		if !ok {
			continue
		}

		for _, d := range dPerParty {
			amount, ok := num.UintFromString(d.GetAmount(), 10)
			if ok {
				ns.log.Error("Failed to create amount string", logging.String("string", d.GetAmount()))
				continue
			}

			stakedTotal.AddSum(amount)
		}
	}

	return stakedTotal.String()
}

func (ns *Node) nodeProtoFromInternal(n node) *pb.Node {
	stakedByOperator := num.NewUint(0)
	stakedByDelegates := num.NewUint(0)

	delegations := make([]*pb.Delegation, 0, len(n.delegationsPerEpochPerParty))

	for _, dPerParty := range n.delegationsPerEpochPerParty {
		for _, d := range dPerParty {
			delegation := d
			delegations = append(delegations, &delegation)

			amount, ok := num.UintFromString(d.GetAmount(), 10)
			if ok {
				ns.log.Error("Failed to create amount string", logging.String("string", d.GetAmount()))
				continue
			}

			// If party is equal the node public key we assume this is operator
			if d.GetParty() == n.n.GetPubKey() {
				stakedByOperator.Add(stakedByOperator, amount)
			} else {
				stakedByDelegates.Add(stakedByDelegates, amount)
			}
		}
	}

	stakedTotal := num.Sum(stakedByOperator, stakedByDelegates)

	// @TODO finish these fields
	// PendingStake string
	// Epoch data

	return &pb.Node{
		Id:                n.n.GetId(),
		PubKey:            n.n.GetPubKey(),
		InfoUrl:           n.n.GetInfoUrl(),
		Location:          n.n.GetLocation(),
		Status:            n.n.GetStatus(),
		StakedByOperator:  stakedByOperator.String(),
		StakedByDelegates: stakedByDelegates.String(),
		StakedTotal:       stakedTotal.String(),

		Delagations: delegations,
	}
}
