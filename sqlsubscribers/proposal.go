package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type ProposalEvent interface {
	events.Event
	ProposalID() string
	PartyID() string
	Proposal() vega.Proposal
}

type ProposalStore interface {
	Add(context.Context, entities.Proposal) error
}

type Proposal struct {
	store    ProposalStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewProposal(
	store ProposalStore,
	log *logging.Logger,
) *Proposal {
	ps := &Proposal{
		store: store,
		log:   log,
	}
	return ps
}

func (ps *Proposal) Type() events.Type {
	return events.ProposalEvent
}

func (rs *Proposal) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		rs.vegaTime = event.Time()
	case ProposalEvent:
		rs.consume(event)
	default:
		rs.log.Panic("Unknown event type in rewards subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (ps *Proposal) consume(event ProposalEvent) {
	protoProposal := event.Proposal()
	proposal, err := entities.ProposalFromProto(&protoProposal)

	// The timestamp in the proto proposal is the time of the initial proposal, not any update
	proposal.VegaTime = ps.vegaTime
	if err != nil {
		ps.log.Error("unable to parse proposal", logging.Error(err))
		return
	}

	if err := ps.store.Add(context.Background(), proposal); err != nil {
		ps.log.Error("Error adding proposal", logging.Error(err))
	}
}
