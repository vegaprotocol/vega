// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
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
