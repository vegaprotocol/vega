// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package services

import (
	"context"
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/subscribers"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var ErrMissingProposalOrPartyFilter = errors.New("missing proposal or party filter")

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
	for k := range partyVotes {
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
