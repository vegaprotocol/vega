package services

import (
	"context"
	"errors"
	"sync"

	vegapb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
)

var (
	ErrMissingProposalOrPartyFilter = errors.New("missing proposal or party filter")
)

type voteE interface {
	events.Event
	Vote() vegapb.Vote
}

type Votes struct {
	*subscribers.Base
	ctx context.Context

	mu sync.RWMutex
	// map of proposal id -> vote id -> vote
	votes map[string]map[string]vegapb.Vote
	// map of proposer -> set of vote id
	votesPerParty map[string]map[string]struct{}
	ch            chan vegapb.Vote
}

func NewVotes(ctx context.Context) (votes *Votes) {
	defer func() { go votes.consume() }()
	return &Votes{
		Base:          subscribers.NewBase(ctx, 1000, true),
		ctx:           ctx,
		votes:         map[string]map[string]vegapb.Vote{},
		votesPerParty: map[string]map[string]struct{}{},
		ch:            make(chan vegapb.Vote, 100),
	}
}

func (v *Votes) consume() {
	defer func() { close(v.ch) }()
	for {
		select {
		case <-v.Closed():
			return
		case vote, ok := <-v.ch:
			if !ok {
				// cleanup base
				v.Halt()
				// channel is closed
				return
			}
			v.mu.Lock()
			// first add to the proposals maps
			votes, ok := v.votes[vote.ProposalId]
			if !ok {
				votes = map[string]vegapb.Vote{}
				v.votes[vote.ProposalId] = votes
			}
			votes[vote.PartyId] = vote

			// next to the party
			partyVotes, ok := v.votesPerParty[vote.PartyId]
			if !ok {
				partyVotes = map[string]struct{}{}
				v.votesPerParty[vote.PartyId] = partyVotes
			}
			partyVotes[vote.ProposalId] = struct{}{}
			v.mu.Unlock()
		}
	}
}

func (v *Votes) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(voteE); ok {
			v.ch <- ae.Vote()
		}
	}
}

func (v *Votes) List(proposal, party string) ([]*vegapb.Vote, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if len(proposal) > 0 && len(party) > 0 {
		return v.getVotesPerProposalAndParty(proposal, party), nil
	} else if len(party) > 0 {
		return v.getPartyVotes(party), nil
	} else if len(proposal) > 0 {
		return v.getProposalVotes(proposal), nil
	}
	return nil, ErrMissingProposalOrPartyFilter
}

func (v *Votes) getVotesPerProposalAndParty(proposal, party string) []*vegapb.Vote {
	out := []*vegapb.Vote{}
	propVotes, ok := v.votes[proposal]
	if !ok {
		return out
	}

	vote, ok := propVotes[party]
	if ok {
		out = append(out, &vote)
	}

	return out
}

func (v *Votes) getPartyVotes(party string) []*vegapb.Vote {
	partyVotes, ok := v.votesPerParty[party]
	if !ok {
		return nil
	}

	out := make([]*vegapb.Vote, 0, len(partyVotes))
	for k, _ := range partyVotes {
		vote := v.votes[k][party]
		out = append(out, &vote)
	}
	return out
}

func (v *Votes) getProposalVotes(proposal string) []*vegapb.Vote {
	proposalVotes, ok := v.votes[proposal]
	if !ok {
		return nil
	}

	out := make([]*vegapb.Vote, 0, len(proposalVotes))
	for _, v := range proposalVotes {
		v := v
		out = append(out, &v)
	}
	return out
}

func (v *Votes) Types() []events.Type {
	return []events.Type{
		events.VoteEvent,
	}
}
