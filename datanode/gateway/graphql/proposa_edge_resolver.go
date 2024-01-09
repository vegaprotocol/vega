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

package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	types "code.vegaprotocol.io/vega/protos/vega"
	vega "code.vegaprotocol.io/vega/protos/vega"
)

type proposalEdgeResolver VegaResolverRoot

func (r *proposalEdgeResolver) ProposalNode(ctx context.Context, data *v2.GovernanceDataEdge) (ProposalNode, error) {
	if data == nil || data.Node == nil || data.Node.Proposal == nil {
		return nil, ErrInvalidProposal
	}

	if data.GetNode().ProposalType == vega.GovernanceData_TYPE_BATCH {
		return r.BatchProposal(ctx, data.Node)
	}

	return data.Node, nil
}

func (r *proposalEdgeResolver) BatchProposal(ctx context.Context, data *types.GovernanceData) (ProposalNode, error) {
	proposal := data.Proposal

	resolver := (*proposalResolver)(r)

	party, err := resolver.Party(ctx, data)
	if err != nil {
		return nil, err
	}

	votes, err := resolver.Votes(ctx, data)
	if err != nil {
		return nil, err
	}

	return BatchProposal{
		ID:                      &proposal.Id,
		Reference:               proposal.Reference,
		Party:                   party,
		State:                   proposal.State,
		Datetime:                proposal.Timestamp,
		Rationale:               proposal.Rationale,
		Votes:                   votes,
		RejectionReason:         proposal.Reason,
		ErrorDetails:            proposal.ErrorDetails,
		RequiredMajority:        proposal.RequiredMajority,
		RequiredParticipation:   proposal.RequiredParticipation,
		RequiredLpMajority:      proposal.RequiredLiquidityProviderMajority,
		RequiredLpParticipation: proposal.RequiredLiquidityProviderParticipation,
		SubProposals:            data.Proposals,
	}, nil
}
