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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type VoteEvent interface {
	events.Event
	ProposalID() string
	PartyID() string
	Vote() vega.Vote
	Value() vega.Vote_Value
}

type GovernanceService interface {
	AddVote(context.Context, entities.Vote) error
}

type Vote struct {
	subscriber
	store GovernanceService
}

func NewVote(store GovernanceService) *Vote {
	vs := &Vote{
		store: store,
	}
	return vs
}

func (vs *Vote) Types() []events.Type {
	return []events.Type{events.VoteEvent}
}

func (vs *Vote) Push(ctx context.Context, evt events.Event) error {
	return vs.consume(ctx, evt.(VoteEvent))
}

func (vs *Vote) consume(ctx context.Context, event VoteEvent) error {
	protoVote := event.Vote()
	vote, err := entities.VoteFromProto(&protoVote, entities.TxHash(event.TxHash()))

	// The timestamp provided on the vote proto object is from when the vote was first created.
	// It doesn't change when the vote is updated (e.g. with TotalGovernanceTokenWeight et al when
	// the proposal closes.)
	vote.VegaTime = vs.vegaTime

	if err != nil {
		return errors.Wrap(err, "unable to parse vote")
	}

	return errors.Wrap(vs.store.AddVote(ctx, vote), "error adding vote:%w")
}

func (vs *Vote) Name() string {
	return "Vote"
}
