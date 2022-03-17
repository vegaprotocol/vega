package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type VoteEvent interface {
	events.Event
	ProposalID() string
	PartyID() string
	Vote() vega.Vote
	Value() vega.Vote_Value
}

type VoteStore interface {
	Add(context.Context, entities.Vote) error
}

type Vote struct {
	store    VoteStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewVote(
	store VoteStore,
	log *logging.Logger,
) *Vote {
	vs := &Vote{
		store: store,
		log:   log,
	}
	return vs
}

func (vs *Vote) Type() events.Type {
	return events.VoteEvent
}

func (vs *Vote) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		vs.vegaTime = event.Time()
	case VoteEvent:
		vs.consume(event)
	default:
		vs.log.Panic("Unknown event type in vote subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (vs *Vote) consume(event VoteEvent) {
	protoVote := event.Vote()
	vote, err := entities.VoteFromProto(&protoVote)

	if vote.VegaTime != vs.vegaTime {
		vs.log.Error("proposal timestamp does not match current VegaTime",
			logging.Reflect("reward", protoVote))
	}

	if err != nil {
		vs.log.Error("unable to parse vote", logging.Error(err))
		return
	}

	if err := vs.store.Add(context.Background(), vote); err != nil {
		vs.log.Error("Error adding vote", logging.Error(err))
	}
}
