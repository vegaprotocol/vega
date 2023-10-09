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
