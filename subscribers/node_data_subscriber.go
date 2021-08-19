package subscribers

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"
)

// NodeData ...
type NodeData interface {
	FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error)
	GenerateCandlesFromBuffer(marketID string, previousCandlesBuf map[string]types.Candle) error
}

type ValidatorUpdateEvent interface {
	events.Event
	Proto() eventspb.ValidatorUpdate
}

type DelegationBalanceEvent interface {
	events.Event
	Proto() eventspb.DelegationBalanceEvent
}

type EpochUpdateEvent interface {
	events.Event
	Proto() eventspb.EpochEvent
}

type delegation struct {
	nodeID string
	party  string
	amount *num.Uint
	epoch  string
}

type node struct {
	id       string
	pubKey   string
	infoURL  string
	location string

	status types.NodeStatus

	delegationsPerParty map[string]delegation
}

type epoch struct {
	seq       string
	startTime int64
	endTime   int64
	nodeIDs   []string

	delegationsPerNodePerParty map[string]map[string]delegation
}

type NodeDataSub struct {
	*Base
	store CandleStore
	mu    sync.RWMutex

	currentEpoch string

	nodes  map[string]node
	epochs map[string]epoch

	log *logging.Logger
}

func NewNodeDataSub(ctx context.Context, store CandleStore, log *logging.Logger, ack bool) *NodeDataSub {
	sub := &NodeDataSub{
		Base:  NewBase(ctx, 1, ack),
		store: store,
		log:   log,
	}
	return sub
}

func (ns *NodeDataSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	// trade events are batched, we need to lock outside of the loop
	ns.mu.Lock()
	for _, e := range evts {
		switch et := e.(type) {
		case ValidatorUpdateEvent:
			vu := et.Proto()

			ns.nodes[vu.GetVegaPubKey()] = node{
				id:       vu.GetInfoUrl(),
				pubKey:   vu.GetVegaPubKey(),
				infoURL:  vu.GetInfoUrl(),
				location: vu.GetCountry(),
				// For now all nodes are validators
				status: types.NodeStatus_NODE_STATUS_VALIDATOR,
			}
		case EpochUpdateEvent:
			eu := et.Proto()

			seq := strconv.FormatUint(eu.GetSeq(), 10)

			ns.epochs[seq] = epoch{
				seq:       seq,
				startTime: eu.GetStartTime(),
				endTime:   eu.GetEndTime(),
			}

			ns.currentEpoch = seq

		case DelegationBalanceEvent:
			dbe := et.Proto()
			ns.addDelegationToNode(dbe)
			ns.addDelegateToEpoch(dbe)
		default:
			ns.log.Panic("Unknown event type in candles subscriber", logging.String("Type", et.Type().String()))
		}
	}
	ns.mu.Unlock()
}

func (ns *NodeDataSub) addDelegateToEpoch(de eventspb.DelegationBalanceEvent) {
	e, ok := ns.epochs[de.EpochSeq]
	if !ok {
		ns.log.Error("Failed to update event for non existing epoch", logging.String("epoch", de.EpochSeq))
	}

	delegationsPerNodes, ok := e.delegationsPerNodePerParty[de.NodeId]
	if !ok {
		delegationsPerNodes = map[string]delegation{}
	}

	delegationsPerNodes[de.GetParty()] = delegationInternalFromEvent(de)
}

func (ns *NodeDataSub) addDelegationToNode(de eventspb.DelegationBalanceEvent) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	node, ok := ns.nodes[de.GetNodeId()]
	if !ok {
		ns.log.Error("Received delegation balance event for non existing node", logging.String("node_id", de.GetNodeId()))
		return
	}

	node.delegationsPerParty[de.GetParty()] = delegationInternalFromEvent(de)
}

func (ns *NodeDataSub) GetNodeByID(id string) (*types.Node, error) {
	ns.mu.RLock()
	defer ns.mu.RLocker()

	node, ok := ns.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node %s not found", id)
	}

	return nodeProtoFromInternal(node), nil
}

func (ns *NodeDataSub) GetNodes() []*types.Node {
	ns.mu.RLock()
	defer ns.mu.RLocker()

	nodes := make([]*types.Node, len(ns.nodes))
	for _, n := range ns.nodes {
		nodes = append(nodes, nodeProtoFromInternal(n))
	}

	return nodes
}

func (ns *NodeDataSub) GetNodeData() *types.NodeData {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	stakedTotal := num.NewUint(0)

	for _, n := range ns.nodes {
		for _, d := range n.delegationsPerParty {
			stakedTotal.Add(stakedTotal, d.amount)
		}
	}

	var uptime time.Duration
	for _, e := range ns.epochs {
		uptime += time.Unix(0, e.endTime).Sub(time.Unix(0, e.startTime))
	}

	nodesLen := uint32(len(ns.nodes))

	return &types.NodeData{
		StakedTotal:     stakedTotal.String(),
		TotalNodes:      nodesLen,
		ValidatingNodes: nodesLen, // For now this is the same as total nodes
		Uptime:          float32(uptime.Minutes()),
	}
}

func (ns *NodeDataSub) GetEpoch() (*types.Epoch, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	epoch := ns.epochs[ns.currentEpoch]

	e, err := ns.epochProtoFromInternal(epoch)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (ns *NodeDataSub) GetEpochByID(id string) (*types.Epoch, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	epoch, ok := ns.epochs[id]
	if !ok {
		return nil, fmt.Errorf("epoch %s not found", id)
	}

	e, err := ns.epochProtoFromInternal(epoch)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (ns *NodeDataSub) Types() []events.Type {
	return []events.Type{
		events.ValidatorUpdateEvent,
		events.DelegationBalanceEvent,
		events.EpochUpdate,
	}
}

func (ns *NodeDataSub) epochProtoFromInternal(e epoch) (*types.Epoch, error) {
	seq, err := strconv.ParseUint(e.seq, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uint from %s: %w", e.seq, err)
	}

	validators := make([]*types.Node, len(e.nodeIDs))
	for _, id := range e.nodeIDs {
		n, ok := ns.nodes[id]
		if !ok {
			continue
		}

		validators = append(validators, nodeProtoFromInternal(n))
	}

	delegations := make([]*types.Delegation, len(e.delegationsPerNodePerParty))
	for _, delegationPerParty := range e.delegationsPerNodePerParty {
		for _, delegation := range delegationPerParty {
			delegations = append(delegations, delegationProtoFromInternal(delegation))
		}
	}

	return &types.Epoch{
		Seq: seq,
		Timestamps: &types.EpochTimestamps{
			StartTime: e.startTime,
			EndTime:   e.endTime,
			// @TODO - add those later
			// FirstBlock: uint64,
			// LastBlock: uint64,
		},
		Validators:  validators,
		Delegations: delegations,
	}, nil
}

func delegationInternalFromEvent(de eventspb.DelegationBalanceEvent) delegation {
	return delegation{
		nodeID: de.NodeId,
		party:  de.Party,
		amount: num.NewUint(de.GetAmount()),
		epoch:  de.EpochSeq,
	}
}

func delegationProtoFromInternal(d delegation) *types.Delegation {
	return &types.Delegation{
		NodeId:   d.nodeID,
		Party:    d.party,
		Amount:   d.amount.String(),
		EpochSeq: d.epoch,
	}
}

func nodeProtoFromInternal(n node) *types.Node {
	stakedTotal := num.NewUint(0)
	stakedByOperator := num.NewUint(0)
	stakedByDelegates := num.NewUint(0)
	delegations := make([]*types.Delegation, len(n.delegationsPerParty))

	for _, d := range n.delegationsPerParty {
		delegations = append(delegations, delegationProtoFromInternal(d))

		// If party is equal the node public key we assume this is operator
		if d.party == n.pubKey {
			stakedByOperator.Add(stakedByOperator, d.amount)
		} else {
			stakedByDelegates.Add(stakedByDelegates, d.amount)
		}
	}

	stakedTotal.Add(stakedByOperator, stakedByDelegates)

	// @TODO finish these fields
	// PendingStake string
	// Epoch data

	return &types.Node{
		Id:                n.id,
		PubKey:            n.pubKey,
		InfoUrl:           n.infoURL,
		Location:          n.location,
		Status:            n.status,
		StakedByOperator:  stakedByOperator.String(),
		StakedByDelegates: stakedByDelegates.String(),
		StakedTotal:       stakedTotal.String(),

		Delagations: delegations,
	}
}
