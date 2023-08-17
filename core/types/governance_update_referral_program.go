// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
