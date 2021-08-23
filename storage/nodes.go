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
		nodes:  make(map[string]node),
		log:    log,
		Config: c,
	}
}

// ReloadConf update the internal conf of the market
func (n *Node) ReloadConf(cfg Config) {
	n.log.Info("reloading configuration")
	if n.log.GetLevel() != cfg.Level.Get() {
		n.log.Info("updating log level",
			logging.String("old", n.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		n.log.SetLevel(cfg.Level.Get())
	}

	n.Config = cfg
}

func (v *Node) AddNode(n pb.Node) {
	v.mut.Lock()
	defer v.mut.Unlock()

	v.nodes[n.GetPubKey()] = node{
		n:                           n,
		delegationsPerEpochPerParty: make(map[string]map[string]pb.Delegation),
	}
}

func (v *Node) AddDelegation(de pb.Delegation) {
	v.mut.Lock()
	defer v.mut.Unlock()

	node, ok := v.nodes[de.GetNodeId()]
	if !ok {
		v.log.Error("Received delegation balance event for non existing node", logging.String("node_id", de.GetNodeId()))
		return
	}

	if _, ok := node.delegationsPerEpochPerParty[de.GetEpochSeq()]; !ok {
		node.delegationsPerEpochPerParty[de.GetEpochSeq()] = map[string]pb.Delegation{}
	}

	node.delegationsPerEpochPerParty[de.GetEpochSeq()][de.GetParty()] = de
}

func (v *Node) GetByID(id string) (*pb.Node, error) {
	v.mut.RLock()
	defer v.mut.RUnlock()

	node, ok := v.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}

	return v.nodeProtoFromInternal(node), nil
}

func (v *Node) GetAll() []*pb.Node {
	v.mut.RLock()
	defer v.mut.RUnlock()

	nodes := make([]*pb.Node, 0, len(v.nodes))
	for _, n := range v.nodes {
		nodes = append(nodes, v.nodeProtoFromInternal(n))
	}

	return nodes
}

func (v *Node) GetAllIDs() []string {
	v.mut.RLock()
	defer v.mut.RUnlock()

	ids := make([]string, 0, len(v.nodes))
	for _, n := range v.nodes {
		ids = append(ids, n.n.GetPubKey())
	}

	return ids
}

func (v *Node) GetTotalNodesNumber() int {
	v.mut.RLock()
	defer v.mut.RUnlock()

	return len(v.nodes)
}

// GetValidatingNodesNumber - for now this is the same as total nodes
func (v *Node) GetValidatingNodesNumber() int {
	v.mut.RLock()
	defer v.mut.RUnlock()

	return len(v.nodes)
}

// GetStakedTotal returns total stake accross all nodes per epoch.
// Returns 0 if epoch not exists.
func (v *Node) GetStakedTotal(epochID string) string {
	v.mut.RLock()
	defer v.mut.RUnlock()

	stakedTotal := num.NewUint(0)

	for _, n := range v.nodes {
		dPerParty, ok := n.delegationsPerEpochPerParty[epochID]
		if !ok {
			continue
		}

		for _, d := range dPerParty {
			amount, ok := num.UintFromString(d.GetAmount(), 10)
			if ok {
				v.log.Error("Failed to create amount string", logging.String("string", d.GetAmount()))
				continue
			}

			stakedTotal.Add(stakedTotal, amount)
		}
	}

	return stakedTotal.String()
}

func (v *Node) nodeProtoFromInternal(n node) *pb.Node {
	stakedTotal := num.NewUint(0)
	stakedByOperator := num.NewUint(0)
	stakedByDelegates := num.NewUint(0)

	delegations := make([]*pb.Delegation, 0, len(n.delegationsPerEpochPerParty))

	for _, dPerParty := range n.delegationsPerEpochPerParty {
		for _, d := range dPerParty {
			delegation := d
			delegations = append(delegations, &delegation)

			amount, ok := num.UintFromString(d.GetAmount(), 10)
			if ok {
				v.log.Error("Failed to create amount string", logging.String("string", d.GetAmount()))
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

	stakedTotal.Add(stakedByOperator, stakedByDelegates)

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
