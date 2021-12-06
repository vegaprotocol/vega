package storage

import (
	"fmt"
	"strconv"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	pb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

type nodeScore struct {
	score           string
	normalisedScore string
}

type node struct {
	n pb.Node

	delegationsPerEpochPerParty map[string]map[string]pb.Delegation
	scoresPerEpoch              map[string]nodeScore
}

type keyRotation struct {
	nodeId      string
	oldPubKey   string
	newPubKey   string
	blockHeight uint64
}

type Node struct {
	Config

	nodes                  map[string]node
	pubKeyrotationsPerNode map[string][]keyRotation
	mut                    sync.RWMutex

	log *logging.Logger
}

func NewNode(log *logging.Logger, c Config) *Node {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &Node{
		nodes:                  map[string]node{},
		pubKeyrotationsPerNode: map[string][]keyRotation{},
		log:                    log,
		Config:                 c,
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
		scoresPerEpoch:              map[string]nodeScore{},
		delegationsPerEpochPerParty: map[string]map[string]pb.Delegation{},
	}
}

func (ns *Node) AddNodeScore(nodeID, epochID, score, normalisedScore string) {
	ns.mut.Lock()
	defer ns.mut.Unlock()

	node, ok := ns.nodes[nodeID]
	if !ok {
		ns.log.Error("Received node score for non existing node", logging.String("node_id", nodeID))
		return
	}

	node.scoresPerEpoch[epochID] = nodeScore{
		score:           score,
		normalisedScore: normalisedScore,
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

func (ns *Node) GetByID(id, epochID string) (*pb.Node, error) {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	node, ok := ns.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}

	return ns.nodeProtoFromInternal(node, epochID), nil
}

// GetAll returns all nodes per epoch
func (ns *Node) GetAll(epochID string) []*pb.Node {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	nodes := make([]*pb.Node, 0, len(ns.nodes))
	for _, n := range ns.nodes {
		nodes = append(nodes, ns.nodeProtoFromInternal(n, epochID))
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

func (ns *Node) PublickKeyChanged(nodeID string, oldPubKey string, newPubKey string, blockHeight uint64) {
	ns.mut.Lock()
	defer ns.mut.Unlock()

	node, ok := ns.nodes[nodeID]
	if !ok {
		ns.log.Error("Received public key change for non existing node", logging.String("node_id", nodeID))
		return
	}

	// update public key in node
	node.n.PubKey = newPubKey
	ns.nodes[nodeID] = node

	// add to pub key rotations history
	if _, ok := ns.pubKeyrotationsPerNode[nodeID]; !ok {
		ns.pubKeyrotationsPerNode[nodeID] = []keyRotation{}
	}

	ns.pubKeyrotationsPerNode[nodeID] = append(ns.pubKeyrotationsPerNode[nodeID], keyRotation{
		nodeId:      nodeID,
		oldPubKey:   oldPubKey,
		newPubKey:   newPubKey,
		blockHeight: blockHeight,
	})
}

func (ns *Node) GetAllPubKeyRotations() []*protoapi.KeyRotation {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	rotations := make([]*protoapi.KeyRotation, 0, len(ns.pubKeyrotationsPerNode))
	for _, rts := range ns.pubKeyrotationsPerNode {
		for _, r := range rts {
			rotations = append(rotations, keyRotationProtoFromInternal(r))
		}
	}
	return rotations
}

func (ns *Node) GetPubKeyRotationsPerNode(nodeID string) []*protoapi.KeyRotation {
	ns.mut.RLock()
	defer ns.mut.RUnlock()

	internalRotations, ok := ns.pubKeyrotationsPerNode[nodeID]
	if !ok {
		return []*protoapi.KeyRotation{}
	}

	rotations := make([]*protoapi.KeyRotation, 0, len(internalRotations))
	for _, r := range internalRotations {
		rotations = append(rotations, keyRotationProtoFromInternal(r))
	}
	return rotations
}

func keyRotationProtoFromInternal(kr keyRotation) *protoapi.KeyRotation {
	return &protoapi.KeyRotation{
		NodeId:      kr.nodeId,
		NewPubKey:   kr.newPubKey,
		OldPubKey:   kr.oldPubKey,
		BlockHeight: kr.blockHeight,
	}
}

func (ns *Node) nodeProtoFromInternal(n node, epochID string) *pb.Node {
	stakedByOperator := num.NewUint(0)
	stakedByDelegates := num.NewUint(0)
	pendingStake := num.NewUint(0)
	pendingStakeSign := true // true = pos, false = neg

	var delegations []*pb.Delegation

	amounts := map[string]*num.Uint{}
	if dPerParty, ok := n.delegationsPerEpochPerParty[epochID]; ok {
		for party, d := range dPerParty {
			delegation := d
			delegations = append(delegations, &delegation)

			amount, ok := num.UintFromString(d.GetAmount(), 10)
			if ok {
				ns.log.Error("Failed to create amount string", logging.String("string", d.GetAmount()))
				continue
			}

			amounts[party] = amount

			// If party is equal the node public key we assume this is operator
			if d.GetParty() == n.n.GetPubKey() {
				stakedByOperator.Add(stakedByOperator, amount)
			} else {
				stakedByDelegates.Add(stakedByDelegates, amount)
			}

		}
	}

	// now we try to get the next epoch so we could calculate the pending stake
	epochSeq, err := strconv.ParseUint(epochID, 10, 64)
	if err != nil {
		ns.log.Error("could not convert back epochID to uint", logging.Error(err))
		return nil
	}

	// may be nil but that's fine
	nextDPerParty := n.delegationsPerEpochPerParty[fmt.Sprintf("%d", epochSeq+1)]
	// compute pending now
	for party, nextD := range nextDPerParty {
		nextAmount, ok := num.UintFromString(nextD.GetAmount(), 10)
		if ok {
			ns.log.Error("Failed to create amount string", logging.String("string", nextD.GetAmount()))
			continue
		}

		amount, ok := amounts[party]
		if !ok {
			amount = num.Zero()
		}

		// add to the pending diff then
		if nextAmount.GT(amount) {
			pendingStakeSign = addToPending(pendingStakeSign, pendingStake, num.Zero().Sub(nextAmount, amount))
		} else {
			pendingStakeSign = subFromPending(pendingStakeSign, pendingStake, num.Zero().Sub(amount, nextAmount))
		}
	}

	stakedTotal := num.Sum(stakedByOperator, stakedByDelegates)
	pendingStakeString := "0"
	if !pendingStake.IsZero() {
		if pendingStakeSign {
			pendingStakeString = fmt.Sprintf("+%s", pendingStake.String())
		} else {
			pendingStakeString = fmt.Sprintf("-%s", pendingStake.String())
		}
	}

	// @TODO finish these fields
	// PendingStake string
	// Epoch data
	node := &pb.Node{
		Id:                n.n.GetId(),
		PubKey:            n.n.GetPubKey(),
		TmPubKey:          n.n.GetTmPubKey(),
		EthereumAdddress:  n.n.GetEthereumAdddress(),
		InfoUrl:           n.n.GetInfoUrl(),
		Location:          n.n.GetLocation(),
		Status:            n.n.GetStatus(),
		StakedByOperator:  stakedByOperator.String(),
		StakedByDelegates: stakedByDelegates.String(),
		StakedTotal:       stakedTotal.String(),
		PendingStake:      pendingStakeString,
		Name:              n.n.GetName(),
		AvatarUrl:         n.n.GetAvatarUrl(),
		Delegations:       delegations,
	}

	if sc, ok := n.scoresPerEpoch[epochID]; ok {
		node.Score = sc.score
		node.NormalisedScore = sc.normalisedScore
	}

	return node
}

func addToPending(sign bool, pending, amount *num.Uint) bool {
	if sign {
		// positive just add to it
		pending.Add(pending, amount)
		return sign
	}
	if pending.GT(amount) {
		pending.Sub(pending, amount)
		return sign
	}

	pending.Sub(amount, pending)
	return !sign
}

func subFromPending(sign bool, pending, amount *num.Uint) bool {
	if !sign {
		pending.Add(pending, amount)
		return sign
	}
	if pending.GT(amount) {
		pending.Sub(pending, amount)
		return sign
	}

	pending.Sub(amount, pending)
	return !sign
}
