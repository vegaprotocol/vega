package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
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

func (rs *Reward) Types() []events.Type {
	return []events.Type{events.RewardPayoutEvent}
}

func (rs *Reward) Push(evt events.Event) error {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		rs.vegaTime = event.Time()
	case RewardPayoutEvent:
		return rs.consume(event)
	default:
		return errors.Errorf("unknown event type %s", event.Type().String())
	}

	return nil
}

func (rs *Reward) consume(event RewardPayoutEvent) error {
	protoRewardPayoutEvent := event.RewardPayoutEvent()
	reward, err := entities.RewardFromProto(protoRewardPayoutEvent)
	if err != nil {
		return errors.Wrap(err, "unable to parse reward")
	}

	if reward.VegaTime != rs.vegaTime {
		return errors.Errorf("reward timestamp does not match current VegaTime. Reward:%v",
			protoRewardPayoutEvent)
	}

	return errors.Wrap(rs.store.Add(context.Background(), reward), "error adding reward payout")
}
