package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type ValidatorUpdateEvent interface {
	events.Event
	ValidatorUpdate() eventspb.ValidatorUpdate
}

type ValidatorRankingScoreEvent interface {
	events.Event
	ValidatorRankingEvent() eventspb.ValidatorRankingEvent
}

type ValidatorRewardScoreEvent interface {
	events.Event
	ValidatorScoreEvent() eventspb.ValidatorScoreEvent
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers NodeStore
type NodeStore interface {
	UpsertNode(context.Context, *entities.Node) error
	UpsertRanking(context.Context, *entities.RankingScore, *entities.RankingScoreAux) error
	UpsertScore(context.Context, *entities.RewardScore, *entities.RewardScoreAux) error
	UpdatePublicKey(context.Context, *entities.KeyRotation) error
	AddNodeAnnoucedEvent(context.Context, entities.NodeID, *entities.ValidatorUpdateAux) error
}

type Node struct {
	subscriber
	store NodeStore
	log   *logging.Logger
}

func NewNode(store NodeStore, log *logging.Logger) *Node {
	return &Node{
		store: store,
		log:   log,
	}
}

func (_ *Node) Types() []events.Type {
	return []events.Type{events.ValidatorUpdateEvent, events.ValidatorRankingEvent, events.ValidatorScoreEvent, events.KeyRotationEvent}
}

func (n *Node) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case ValidatorUpdateEvent:
		return n.consumeUpdate(ctx, e)
	case ValidatorRankingScoreEvent:
		return n.consumeRankingScore(ctx, e)
	case ValidatorRewardScoreEvent:
		return n.consumeRewardScore(ctx, e)
	case KeyRotationEvent:
		return n.consumeKeyRotation(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (n *Node) consumeUpdate(ctx context.Context, event ValidatorUpdateEvent) error {
	node, aux, err := entities.NodeFromValidatorUpdateEvent(event.ValidatorUpdate(), n.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting validator update event proto to database entity failed")
	}

	if err := errors.Wrap(n.store.UpsertNode(ctx, &node), "inserting node to SQL store failed"); err != nil {
		return err
	}
	return errors.Wrap(n.store.AddNodeAnnoucedEvent(ctx, node.ID, &aux), "inserting node to SQL store failed")
}

func (n *Node) consumeRankingScore(ctx context.Context, event ValidatorRankingScoreEvent) error {
	ranking, aux, err := entities.RankingScoreFromRankingEvent(event.ValidatorRankingEvent(), n.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting ranking score event proto to database entity failed")
	}

	return errors.Wrap(n.store.UpsertRanking(ctx, &ranking, &aux), "inserting ranking score to SQL store failed")
}

func (n *Node) consumeRewardScore(ctx context.Context, event ValidatorRewardScoreEvent) error {
	reward, aux, err := entities.RewardScoreFromScoreEvent(event.ValidatorScoreEvent(), n.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting reward score event proto to database entity failed")
	}

	return errors.Wrap(n.store.UpsertScore(ctx, &reward, &aux), "inserting reward score to SQL store failed")
}

func (n *Node) consumeKeyRotation(ctx context.Context, event KeyRotationEvent) error {
	key_rotation := event.KeyRotation()
	record, err := entities.KeyRotationFromProto(&key_rotation, n.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting key rotation proto to database entity failed")
	}

	return errors.Wrap(n.store.UpdatePublicKey(ctx, record), "Updating public key to SQL store failed")
}
