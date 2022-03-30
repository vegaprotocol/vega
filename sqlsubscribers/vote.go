package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
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

func (vs *Vote) Types() []events.Type {
	return []events.Type{events.VoteEvent}
}

func (vs *Vote) Push(evt events.Event) error {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		vs.vegaTime = event.Time()
	case VoteEvent:
		return vs.consume(event)
	default:
		return errors.Errorf("unknown event type %s", event.Type().String())
	}

	return nil
}

func (vs *Vote) consume(event VoteEvent) error {
	protoVote := event.Vote()
	vote, err := entities.VoteFromProto(&protoVote)

	// The timestamp provided on the vote proto object is from when the vote was first created.
	// It doesn't change when the vote is updated (e.g. with TotalGovernanceTokenWeight et al when
	// the proposal closes.)
	vote.VegaTime = vs.vegaTime

	if err != nil {
		return errors.Wrap(err, "unable to parse vote")
	}

	return errors.Wrap(vs.store.Add(context.Background(), vote), "error adding vote:%w")
}
