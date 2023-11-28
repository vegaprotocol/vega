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

package types

import (
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type BatchProposalSubmission struct {
	// Proposal reference
	Reference string
	// Proposal configuration and the actual change that is meant to be executed when proposal is enacted
	Terms *BatchProposalTerms
	// Rationale behind the proposal change.
	Rationale *ProposalRationale
}

func (p BatchProposalSubmission) IntoProto() *commandspb.BatchProposalSubmission {
	var terms *vegapb.BatchProposalTerms
	if p.Terms != nil {
		terms = p.Terms.IntoProto()
	}
	return &commandspb.BatchProposalSubmission{
		Reference: p.Reference,
		Terms:     terms,
		Rationale: &vegapb.ProposalRationale{
			Description: p.Rationale.Description,
			Title:       p.Rationale.Title,
		},
	}
}

// TODO karel - make this batch proposal
// func BatchProposalSubmissionFromProposal(p *Proposal) *BatchProposalSubmission {
// 	return &BatchProposalSubmission{
// 		Reference: p.Reference,
// 		Terms:     p.Terms,
// 		Rationale: p.Rationale,
// 	}
// }

func NewBatchProposalSubmissionFromProto(p *commandspb.BatchProposalSubmission) (*BatchProposalSubmission, error) {
	var pterms *BatchProposalTerms
	if p.Terms != nil {
		var err error
		pterms, err = BatchProposalTermsFromProto(p.Terms)
		if err != nil {
			return nil, err
		}
	}
	return &BatchProposalSubmission{
		Reference: p.Reference,
		Terms:     pterms,
		Rationale: &ProposalRationale{
			Description: p.Rationale.Description,
			Title:       p.Rationale.Title,
		},
	}, nil
}
