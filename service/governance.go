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

package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type ProposalStore interface {
	Add(ctx context.Context, p entities.Proposal) error
	GetByID(ctx context.Context, id string) (entities.Proposal, error)
	GetByReference(ctx context.Context, ref string) (entities.Proposal, error)
	Get(ctx context.Context, inState *entities.ProposalState, partyIDStr *string, proposalType *entities.ProposalType,
		pagination entities.CursorPagination) ([]entities.Proposal, entities.PageInfo, error)
}

type VoteStore interface {
	Add(ctx context.Context, v entities.Vote) error
	GetYesVotesForProposal(ctx context.Context, proposalIDStr string) ([]entities.Vote, error)
	GetNoVotesForProposal(ctx context.Context, proposalIDStr string) ([]entities.Vote, error)
	GetByParty(ctx context.Context, partyIDStr string) ([]entities.Vote, error)
	GetByPartyConnection(ctx context.Context, partyIDStr string, pagination entities.CursorPagination) ([]entities.Vote, entities.PageInfo, error)
	Get(ctx context.Context, proposalID, partyID *string, value *entities.VoteValue) ([]entities.Vote, error)
}

type Governance struct {
	pStore    ProposalStore
	vStore    VoteStore
	log       *logging.Logger
	pObserver utils.Observer[entities.Proposal]
	vObserver utils.Observer[entities.Vote]
}

func NewGovernance(pStore ProposalStore, vStore VoteStore, log *logging.Logger) *Governance {
	return &Governance{
		pStore:    pStore,
		vStore:    vStore,
		pObserver: utils.NewObserver[entities.Proposal]("proposal", log, 0, 0),
		vObserver: utils.NewObserver[entities.Vote]("vote", log, 0, 0),
	}
}

func (g *Governance) AddProposal(ctx context.Context, p entities.Proposal) error {
	err := g.pStore.Add(ctx, p)
	if err != nil {
		return err
	}
	g.pObserver.Notify([]entities.Proposal{p})
	return nil
}

func (g *Governance) GetProposalByID(ctx context.Context, id string) (entities.Proposal, error) {
	return g.pStore.GetByID(ctx, id)
}

func (g *Governance) GetProposalByReference(ctx context.Context, ref string) (entities.Proposal, error) {
	return g.pStore.GetByReference(ctx, ref)
}

func (g *Governance) GetProposals(ctx context.Context, inState *entities.ProposalState, partyID *string, proposalType *entities.ProposalType,
	pagination entities.CursorPagination) ([]entities.Proposal, entities.PageInfo, error) {
	return g.pStore.Get(ctx, inState, partyID, proposalType, pagination)
}

func (g *Governance) ObserveProposals(ctx context.Context, retries int, partyID *string) (<-chan []entities.Proposal, uint64) {
	ch, ref := g.pObserver.Observe(ctx,
		retries,
		func(o entities.Proposal) bool { return partyID == nil || o.PartyID.String() == *partyID })
	return ch, ref
}

func (g *Governance) AddVote(ctx context.Context, v entities.Vote) error {
	err := g.vStore.Add(ctx, v)
	if err != nil {
		return err
	}
	g.vObserver.Notify([]entities.Vote{v})
	return nil
}

func (g *Governance) GetYesVotesForProposal(ctx context.Context, proposalID string) ([]entities.Vote, error) {
	return g.vStore.GetYesVotesForProposal(ctx, proposalID)
}

func (g *Governance) GetNoVotesForProposal(ctx context.Context, proposalID string) ([]entities.Vote, error) {
	return g.vStore.GetNoVotesForProposal(ctx, proposalID)
}

func (g *Governance) GetVotesByParty(ctx context.Context, partyID string) ([]entities.Vote, error) {
	return g.vStore.GetByParty(ctx, partyID)
}

func (p *Governance) GetByPartyConnection(ctx context.Context, partyID string, pagination entities.CursorPagination) ([]entities.Vote, entities.PageInfo, error) {
	return p.vStore.GetByPartyConnection(ctx, partyID, pagination)
}

func (g *Governance) GetVotes(ctx context.Context, proposalID, partyID *string, value *entities.VoteValue) ([]entities.Vote, error) {
	return g.vStore.Get(ctx, proposalID, partyID, value)
}

func (g *Governance) ObservePartyVotes(ctx context.Context, retries int, partyID string) (<-chan []entities.Vote, uint64) {
	ch, ref := g.vObserver.Observe(ctx,
		retries,
		func(o entities.Vote) bool { return o.PartyID.String() == partyID })
	return ch, ref
}

func (g *Governance) ObserveProposalVotes(ctx context.Context, retries int, proposalID string) (<-chan []entities.Vote, uint64) {
	ch, ref := g.vObserver.Observe(ctx,
		retries,
		func(o entities.Vote) bool { return o.PartyID.String() == proposalID })
	return ch, ref
}
