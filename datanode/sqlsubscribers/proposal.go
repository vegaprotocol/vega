// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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
	log   *logging.Logger
}

func NewProposal(
	store proposalAdder,
	log *logging.Logger,
) *Proposal {
	ps := &Proposal{
		store: store,
		log:   log,
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
	proposal, err := entities.ProposalFromProto(&protoProposal)

	// The timestamp in the proto proposal is the time of the initial proposal, not any update
	proposal.VegaTime = ps.vegaTime
	if err != nil {
		return errors.Wrap(err, "unable to parse proposal")
	}

	return errors.Wrap(ps.store.AddProposal(ctx, proposal), "error adding proposal")
}
