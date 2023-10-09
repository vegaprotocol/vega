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
	"fmt"

	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateReferralProgram struct {
	UpdateReferralProgram *UpdateReferralProgram
}

func (a ProposalTermsUpdateReferralProgram) String() string {
	return fmt.Sprintf(
		"updateReferralProgram(%s)",
		stringer.ReflectPointerToString(a.UpdateReferralProgram),
	)
}

func (a ProposalTermsUpdateReferralProgram) IntoProto() *vegapb.ProposalTerms_UpdateReferralProgram {
	return &vegapb.ProposalTerms_UpdateReferralProgram{
		UpdateReferralProgram: a.UpdateReferralProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateReferralProgram) isPTerm() {}

func (a ProposalTermsUpdateReferralProgram) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateReferralProgram) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateReferralProgram
}

func (a ProposalTermsUpdateReferralProgram) DeepClone() proposalTerm {
	if a.UpdateReferralProgram == nil {
		return &ProposalTermsUpdateReferralProgram{}
	}
	return &ProposalTermsUpdateReferralProgram{
		UpdateReferralProgram: a.UpdateReferralProgram.DeepClone(),
	}
}

func NewUpdateReferralProgramProposalFromProto(p *vegapb.ProposalTerms_UpdateReferralProgram) (*ProposalTermsUpdateReferralProgram, error) {
	return &ProposalTermsUpdateReferralProgram{
		UpdateReferralProgram: NewUpdateReferralProgramFromProto(p.UpdateReferralProgram),
	}, nil
}

type UpdateReferralProgram struct {
	Changes *ReferralProgram
}

func (p UpdateReferralProgram) IntoProto() *vegapb.UpdateReferralProgram {
	return &vegapb.UpdateReferralProgram{
		Changes: p.Changes.IntoProto(),
	}
}

func (p UpdateReferralProgram) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.ReflectPointerToString(p.Changes),
	)
}

func (p UpdateReferralProgram) DeepClone() *UpdateReferralProgram {
	if p.Changes == nil {
		return &UpdateReferralProgram{}
	}
	return &UpdateReferralProgram{
		Changes: p.Changes.DeepClone(),
	}
}

func NewUpdateReferralProgramFromProto(p *vegapb.UpdateReferralProgram) *UpdateReferralProgram {
	if p == nil {
		return &UpdateReferralProgram{}
	}

	return &UpdateReferralProgram{
		Changes: NewReferralProgramFromProto(p.Changes),
	}
}
