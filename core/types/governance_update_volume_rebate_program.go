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
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateVolumeRebateProgram struct {
	UpdateVolumeRebateProgram *UpdateVolumeRebateProgram
}

func (a ProposalTermsUpdateVolumeRebateProgram) String() string {
	return fmt.Sprintf(
		"updateVolumeRebatetProgram(%s)",
		stringer.PtrToString(a.UpdateVolumeRebateProgram),
	)
}

func (a ProposalTermsUpdateVolumeRebateProgram) isPTerm() {}

func (a ProposalTermsUpdateVolumeRebateProgram) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateVolumeRebateProgram{
		UpdateVolumeRebateProgram: a.UpdateVolumeRebateProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateVolumeRebateProgram) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateVolumeRebateProgram{
		UpdateVolumeRebateProgram: a.UpdateVolumeRebateProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateVolumeRebateProgram) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateVolumeRebateProgram
}

func (a ProposalTermsUpdateVolumeRebateProgram) DeepClone() ProposalTerm {
	if a.UpdateVolumeRebateProgram == nil {
		return &ProposalTermsUpdateVolumeRebateProgram{}
	}
	return &ProposalTermsUpdateVolumeRebateProgram{
		UpdateVolumeRebateProgram: a.UpdateVolumeRebateProgram.DeepClone(),
	}
}

func NewUpdateVolumeRebateProgramProposalFromProto(
	UpdateVolumeRebateProgramProto *vegapb.UpdateVolumeRebateProgram,
) (*ProposalTermsUpdateVolumeRebateProgram, error) {
	return &ProposalTermsUpdateVolumeRebateProgram{
		UpdateVolumeRebateProgram: NewUpdateVolumeRebateProgramFromProto(UpdateVolumeRebateProgramProto),
	}, nil
}

type UpdateVolumeRebateProgram struct {
	Changes *VolumeRebateProgramChanges
}

func (p UpdateVolumeRebateProgram) IntoProto() *vegapb.UpdateVolumeRebateProgram {
	return &vegapb.UpdateVolumeRebateProgram{
		Changes: p.Changes.IntoProto(),
	}
}

func (p UpdateVolumeRebateProgram) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.PtrToString(p.Changes),
	)
}

func (p UpdateVolumeRebateProgram) DeepClone() *UpdateVolumeRebateProgram {
	if p.Changes == nil {
		return &UpdateVolumeRebateProgram{}
	}
	return &UpdateVolumeRebateProgram{
		Changes: p.Changes.DeepClone(),
	}
}

func NewUpdateVolumeRebateProgramFromProto(p *vegapb.UpdateVolumeRebateProgram) *UpdateVolumeRebateProgram {
	if p == nil {
		return &UpdateVolumeRebateProgram{}
	}

	return &UpdateVolumeRebateProgram{
		Changes: NewVolumeRebateProgramChangesFromProto(p.Changes),
	}
}

type VolumeRebateProgramChanges struct {
	ID                       string
	Version                  uint64
	EndOfProgramTimestamp    time.Time
	WindowLength             uint64
	VolumeRebateBenefitTiers []*VolumeRebateBenefitTier
}

func (v VolumeRebateProgramChanges) IntoProto() *vegapb.VolumeRebateProgramChanges {
	benefitTiers := make([]*vegapb.VolumeRebateBenefitTier, 0, len(v.VolumeRebateBenefitTiers))
	for _, tier := range v.VolumeRebateBenefitTiers {
		benefitTiers = append(benefitTiers, &vegapb.VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: tier.MinimumPartyMakerVolumeFraction.String(),
			AdditionalMakerRebate:           tier.AdditionalMakerRebate.String(),
		})
	}

	return &vegapb.VolumeRebateProgramChanges{
		BenefitTiers:          benefitTiers,
		EndOfProgramTimestamp: v.EndOfProgramTimestamp.Unix(),
		WindowLength:          v.WindowLength,
	}
}

func (v VolumeRebateProgramChanges) DeepClone() *VolumeRebateProgramChanges {
	benefitTiers := make([]*VolumeRebateBenefitTier, 0, len(v.VolumeRebateBenefitTiers))
	for _, tier := range v.VolumeRebateBenefitTiers {
		benefitTiers = append(benefitTiers, &VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: tier.MinimumPartyMakerVolumeFraction,
			AdditionalMakerRebate:           tier.AdditionalMakerRebate,
		})
	}

	cpy := VolumeRebateProgramChanges{
		ID:                       v.ID,
		Version:                  v.Version,
		EndOfProgramTimestamp:    v.EndOfProgramTimestamp,
		WindowLength:             v.WindowLength,
		VolumeRebateBenefitTiers: benefitTiers,
	}
	return &cpy
}

func (c VolumeRebateProgramChanges) String() string {
	benefitTierStr := ""
	for i, tier := range c.VolumeRebateBenefitTiers {
		if i > 1 {
			benefitTierStr += ", "
		}
		benefitTierStr += fmt.Sprintf("%d(minimumPartyMakerVolumeFraction(%s), additionalMakerRebate(%s))",
			i,
			tier.MinimumPartyMakerVolumeFraction.String(),
			tier.AdditionalMakerRebate.String(),
		)
	}

	return fmt.Sprintf(
		"endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s)",
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
	)
}

func NewVolumeRebateProgramChangesFromProto(v *vegapb.VolumeRebateProgramChanges) *VolumeRebateProgramChanges {
	if v == nil {
		return &VolumeRebateProgramChanges{}
	}

	benefitTiers := make([]*VolumeRebateBenefitTier, 0, len(v.BenefitTiers))
	for _, tier := range v.BenefitTiers {
		minimumPartyMakerVolumeFraction, _ := num.DecimalFromString(tier.MinimumPartyMakerVolumeFraction)
		additionalMakerRebate, _ := num.DecimalFromString(tier.AdditionalMakerRebate)

		benefitTiers = append(benefitTiers, &VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: minimumPartyMakerVolumeFraction,
			AdditionalMakerRebate:           additionalMakerRebate,
		})
	}

	return &VolumeRebateProgramChanges{
		EndOfProgramTimestamp:    time.Unix(v.EndOfProgramTimestamp, 0),
		WindowLength:             v.WindowLength,
		VolumeRebateBenefitTiers: benefitTiers,
	}
}
