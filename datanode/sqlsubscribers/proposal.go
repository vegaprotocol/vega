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

type ProposalEvent interface {
	events.Event
	ProposalID() string
	PartyID() string
	Proposal() vega.Proposal
}

type proposalAdder interface {
	AddProposal(context.Context, entities.Proposal) error
}

type Proposal struct {
	subscriber
	store proposalAdder
}

func NewProposal(store proposalAdder) *Proposal {
	ps := &Proposal{
		store: store,
	}
	return ps
}

func (ps *Proposal) Types() []events.Type {
	return []events.Type{events.ProposalEvent}
}

func (ps *Proposal) Push(ctx context.Context, evt events.Event) error {
	return ps.consume(ctx, evt.(ProposalEvent))
}

func (ps *Proposal) consume(ctx context.Context, event ProposalEvent) error {
	protoProposal := event.Proposal()
	proposal, err := entities.ProposalFromProto(&protoProposal, entities.TxHash(event.TxHash()))

	// The timestamp in the proto proposal is the time of the initial proposal, not any update
	proposal.VegaTime = ps.vegaTime
	if err != nil {
		return errors.Wrap(err, "unable to parse proposal")
	}

	return errors.Wrap(ps.store.AddProposal(ctx, proposal), "error adding proposal")
}
