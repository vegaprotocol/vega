package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type StakeLinkingEvent interface {
	events.Event
	StakeLinking() eventspb.StakeLinking
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/stake_linking_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers StakeLinkingStore
type StakeLinkingStore interface {
	Upsert(linking *entities.StakeLinking) error
}

type StakeLinking struct {
	store    StakeLinkingStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewStakeLinking(store StakeLinkingStore, log *logging.Logger) *StakeLinking {
	return &StakeLinking{
		store: store,
		log:   log,
	}
}

func (sl *StakeLinking) Types() []events.Type {
	return []events.Type{events.StakeLinkingEvent}
}

func (sl *StakeLinking) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		sl.vegaTime = e.Time()
	case StakeLinkingEvent:
		sl.consume(e)
	}
}

func (sl StakeLinking) consume(event StakeLinkingEvent) {
	stake := event.StakeLinking()
	entity, err := entities.StakeLinkingFromProto(&stake, sl.vegaTime)
	if err != nil {
		sl.log.Error("converting stake linking event to database entitiy failed", logging.Error(err))
		return
	}

	if err = sl.store.Upsert(entity); err != nil {
		sl.log.Error("inserting stake linking event to SQL store failed", logging.Error(err))
	}
}
