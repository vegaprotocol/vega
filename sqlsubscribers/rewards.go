package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type RewardPayoutEvent interface {
	events.Event
	RewardPayoutEvent() eventspb.RewardPayoutEvent
}

type RewardStore interface {
	Add(context.Context, entities.Reward) error
}

type Reward struct {
	store    RewardStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewReward(
	store RewardStore,
	log *logging.Logger,
) *Reward {
	t := &Reward{
		store: store,
		log:   log,
	}
	return t
}

func (rs *Reward) Type() events.Type {
	return events.RewardPayoutEvent
}

func (rs *Reward) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		rs.vegaTime = event.Time()
	case RewardPayoutEvent:
		rs.consume(event)
	default:
		rs.log.Panic("Unknown event type in rewards subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (rs *Reward) consume(event RewardPayoutEvent) {
	protoRewardPayoutEvent := event.RewardPayoutEvent()
	reward, err := entities.RewardFromProto(protoRewardPayoutEvent)
	if err != nil {
		rs.log.Error("unable to parse reward", logging.Error(err))
	}

	if reward.VegaTime != rs.vegaTime {
		rs.log.Error("reward timestamp does not match current VegaTime",
			logging.Reflect("reward", protoRewardPayoutEvent))
	}

	if err := rs.store.Add(context.Background(), reward); err != nil {
		rs.log.Error("Error adding reward payout", logging.Error(err))
	}
}
