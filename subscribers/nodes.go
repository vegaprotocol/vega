package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	vegapb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type ValidatorUpdateEvent interface {
	events.Event
	Proto() eventspb.ValidatorUpdate
}

type ValidatorScoreEvent interface {
	events.Event
	Proto() eventspb.ValidatorScoreEvent
}

type ValidatorRankingEvent interface {
	events.Event
	Proto() eventspb.ValidatorRankingEvent
}

type KeyRotationEvent interface {
	events.Event
	Proto() eventspb.KeyRotation
}

type NodeStore interface {
	AddNode(types.Node, bool, uint64)
	AddDelegation(types.Delegation)
	GetAllIDs() []string
	GetByID(id, epochID string) (*vegapb.Node, error)
	AddNodeRewardScore(nodeID, epochID string, scoreData vegapb.RewardScore)
	AddNodeRankingScore(nodeID, epochID string, scoreData vegapb.RankingScore) error
	PublickKeyChanged(nodeID, oldPubKey string, newPubKey string, blockHeight uint64)
}

type NodesSub struct {
	*Base
	nodeStore           NodeStore
	orphanRankingEvents map[string]eventspb.ValidatorRankingEvent
	mu                  sync.Mutex

	log *logging.Logger
}

func NewNodesSub(ctx context.Context, nodeStore NodeStore, log *logging.Logger, ack bool) *NodesSub {
	sub := &NodesSub{
		Base:                NewBase(ctx, 10, ack),
		nodeStore:           nodeStore,
		log:                 log,
		orphanRankingEvents: map[string]eventspb.ValidatorRankingEvent{},
	}

	if sub.isRunning() {
		go sub.loop(ctx)
	}

	return sub
}

func (ns *NodesSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ns.Halt()
			return
		case e := <-ns.ch:
			if ns.isRunning() {
				ns.Push(e...)
			}
		}
	}
}

func validatorStatusToProto(vStatus string) vegapb.ValidatorNodeStatus {
	switch vStatus {
	case "tendermint":
		return vegapb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT
	case "ersatz":
		return vegapb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ
	case "pending":
		return vegapb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_PENDING
	default:
		return vegapb.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_UNSPECIFIED
	}
}

func (ns *NodesSub) addNodeRankingScore(vre eventspb.ValidatorRankingEvent) error {
	ranking := vegapb.RankingScore{
		StakeScore:       vre.GetStakeScore(),
		PerformanceScore: vre.GetPerformanceScore(),
		PreviousStatus:   validatorStatusToProto(vre.PreviousStatus),
		Status:           validatorStatusToProto(vre.NextStatus),
		RankingScore:     vre.GetRankingScore(),
		VotingPower:      vre.GetTmVotingPower(),
	}

	return ns.nodeStore.AddNodeRankingScore(vre.GetNodeId(), vre.GetEpochSeq(), ranking)
}

func (ns *NodesSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	ns.mu.Lock()
	defer ns.mu.Unlock()
	for _, e := range evts {
		switch et := e.(type) {
		case ValidatorUpdateEvent:
			vue := et.Proto()
			ns.nodeStore.AddNode(types.Node{
				Id:               vue.GetNodeId(),
				PubKey:           vue.GetVegaPubKey(),
				TmPubKey:         vue.GetTmPubKey(),
				EthereumAdddress: vue.GetEthereumAddress(),
				InfoUrl:          vue.GetInfoUrl(),
				Location:         vue.GetCountry(),
				Status:           types.NodeStatus_NODE_STATUS_VALIDATOR,
				Name:             vue.GetName(),
				AvatarUrl:        vue.GetAvatarUrl(),
			},
				vue.Added,
				vue.FromEpoch,
			)

			// check if there are any orphaned ranking events that we can send now
			if vre, ok := ns.orphanRankingEvents[vue.GetNodeId()]; ok {
				ns.addNodeRankingScore(vre)
				delete(ns.orphanRankingEvents, vue.GetNodeId())
			}

		case ValidatorScoreEvent:
			vse := et.Proto()
			scores := vegapb.RewardScore{
				RawValidatorScore: vse.GetRawValidatorScore(),
				PerformanceScore:  vse.GetValidatorPerformance(),
				MultisigScore:     vse.GetMultisigScore(),
				ValidatorScore:    vse.GetValidatorScore(),
				NormalisedScore:   vse.GetNormalisedScore(),
				ValidatorStatus:   validatorStatusToProto(vse.ValidatorStatus),
			}
			ns.nodeStore.AddNodeRewardScore(vse.GetNodeId(), vse.GetEpochSeq(), scores)

		case ValidatorRankingEvent:
			vre := et.Proto()
			nodeID := vre.GetNodeId()

			if err := ns.addNodeRankingScore(vre); err != nil {
				ns.log.Info("ranking event received before node was added -- try again later", logging.String("nodeID", nodeID))
				ns.orphanRankingEvents[vre.GetNodeId()] = vre // we'll try to add it in a bit
			}
		case KeyRotationEvent:
			kre := et.Proto()

			ns.nodeStore.PublickKeyChanged(kre.NodeId, kre.OldPubKey, kre.NewPubKey, kre.BlockHeight)
		default:
			ns.log.Panic("Unknown event type in candles subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

func (ns *NodesSub) Types() []events.Type {
	return []events.Type{
		events.ValidatorUpdateEvent,
		events.ValidatorScoreEvent,
		events.ValidatorRankingEvent,
		events.KeyRotationEvent,
	}
}
