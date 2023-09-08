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

type ProposalTermsUpdateVolumeDiscountProgram struct {
	UpdateVolumeDiscountProgram *UpdateVolumeDiscountProgram
}

func (a ProposalTermsUpdateVolumeDiscountProgram) String() string {
	return fmt.Sprintf(
		"updateVolumeDiscountProgram(%s)",
		stringer.ReflectPointerToString(a.UpdateVolumeDiscountProgram),
	)
}

func (a ProposalTermsUpdateVolumeDiscountProgram) IntoProto() *vegapb.ProposalTerms_UpdateVolumeDiscountProgram {
	return &vegapb.ProposalTerms_UpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: a.UpdateVolumeDiscountProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateVolumeDiscountProgram) isPTerm() {}

func (a ProposalTermsUpdateVolumeDiscountProgram) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateVolumeDiscountProgram) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateVolumeDiscountProgram
}

func (a ProposalTermsUpdateVolumeDiscountProgram) DeepClone() proposalTerm {
	if a.UpdateVolumeDiscountProgram == nil {
		return &ProposalTermsUpdateVolumeDiscountProgram{}
	}
	return &ProposalTermsUpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: a.UpdateVolumeDiscountProgram.DeepClone(),
	}
}

func NewUpdateVolumeDiscountProgramProposalFromProto(p *vegapb.ProposalTerms_UpdateVolumeDiscountProgram) (*ProposalTermsUpdateVolumeDiscountProgram, error) {
	return &ProposalTermsUpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: NewUpdateVolumeDiscountProgramFromProto(p.UpdateVolumeDiscountProgram),
	}, nil
}

type UpdateVolumeDiscountProgram struct {
	Changes *VolumeDiscountProgram
}

func (p UpdateVolumeDiscountProgram) IntoProto() *vegapb.UpdateVolumeDiscountProgram {
	return &vegapb.UpdateVolumeDiscountProgram{
		Changes: p.Changes.IntoProto(),
	}
}

func (p UpdateVolumeDiscountProgram) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.ReflectPointerToString(p.Changes),
	)
}

func (p UpdateVolumeDiscountProgram) DeepClone() *UpdateVolumeDiscountProgram {
	if p.Changes == nil {
		return &UpdateVolumeDiscountProgram{}
	}
	return &UpdateVolumeDiscountProgram{
		Changes: p.Changes.DeepClone(),
	}
}

func NewUpdateVolumeDiscountProgramFromProto(p *vegapb.UpdateVolumeDiscountProgram) *UpdateVolumeDiscountProgram {
	if p == nil {
		return &UpdateVolumeDiscountProgram{}
	}

	return &UpdateVolumeDiscountProgram{
		Changes: NewVolumeDiscountProgramFromProto(p.Changes),
	}
}
