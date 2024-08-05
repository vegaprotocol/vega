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

type ProposalTermsUpdateVolumeDiscountProgram struct {
	UpdateVolumeDiscountProgram *UpdateVolumeDiscountProgram
}

func (a ProposalTermsUpdateVolumeDiscountProgram) String() string {
	return fmt.Sprintf(
		"updateVolumeDiscountProgram(%s)",
		stringer.PtrToString(a.UpdateVolumeDiscountProgram),
	)
}

func (a ProposalTermsUpdateVolumeDiscountProgram) isPTerm() {}

func (a ProposalTermsUpdateVolumeDiscountProgram) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: a.UpdateVolumeDiscountProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateVolumeDiscountProgram) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: a.UpdateVolumeDiscountProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateVolumeDiscountProgram) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateVolumeDiscountProgram
}

func (a ProposalTermsUpdateVolumeDiscountProgram) DeepClone() ProposalTerm {
	if a.UpdateVolumeDiscountProgram == nil {
		return &ProposalTermsUpdateVolumeDiscountProgram{}
	}
	return &ProposalTermsUpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: a.UpdateVolumeDiscountProgram.DeepClone(),
	}
}

func NewUpdateVolumeDiscountProgramProposalFromProto(
	updateVolumeDiscountProgramProto *vegapb.UpdateVolumeDiscountProgram,
) (*ProposalTermsUpdateVolumeDiscountProgram, error) {
	return &ProposalTermsUpdateVolumeDiscountProgram{
		UpdateVolumeDiscountProgram: NewUpdateVolumeDiscountProgramFromProto(updateVolumeDiscountProgramProto),
	}, nil
}

type UpdateVolumeDiscountProgram struct {
	Changes *VolumeDiscountProgramChanges
}

func (p UpdateVolumeDiscountProgram) IntoProto() *vegapb.UpdateVolumeDiscountProgram {
	return &vegapb.UpdateVolumeDiscountProgram{
		Changes: p.Changes.IntoProto(),
	}
}

func (p UpdateVolumeDiscountProgram) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.PtrToString(p.Changes),
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
		Changes: NewVolumeDiscountProgramChangesFromProto(p.Changes),
	}
}

type VolumeDiscountProgramChanges struct {
	ID                    string
	Version               uint64
	EndOfProgramTimestamp time.Time
	WindowLength          uint64
	VolumeBenefitTiers    []*VolumeBenefitTier
}

func (v VolumeDiscountProgramChanges) IntoProto() *vegapb.VolumeDiscountProgramChanges {
	benefitTiers := make([]*vegapb.VolumeBenefitTier, 0, len(v.VolumeBenefitTiers))
	for _, tier := range v.VolumeBenefitTiers {
		benefitTiers = append(benefitTiers, &vegapb.VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.String(),
			VolumeDiscountFactors:             tier.VolumeDiscountFactors.IntoDiscountFactorsProto(),
		})
	}

	return &vegapb.VolumeDiscountProgramChanges{
		BenefitTiers:          benefitTiers,
		EndOfProgramTimestamp: v.EndOfProgramTimestamp.Unix(),
		WindowLength:          v.WindowLength,
	}
}

func (v VolumeDiscountProgramChanges) DeepClone() *VolumeDiscountProgramChanges {
	benefitTiers := make([]*VolumeBenefitTier, 0, len(v.VolumeBenefitTiers))
	for _, tier := range v.VolumeBenefitTiers {
		benefitTiers = append(benefitTiers, &VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.Clone(),
			VolumeDiscountFactors:             tier.VolumeDiscountFactors.Clone(),
		})
	}

	cpy := VolumeDiscountProgramChanges{
		ID:                    v.ID,
		Version:               v.Version,
		EndOfProgramTimestamp: v.EndOfProgramTimestamp,
		WindowLength:          v.WindowLength,
		VolumeBenefitTiers:    benefitTiers,
	}
	return &cpy
}

func (c VolumeDiscountProgramChanges) String() string {
	benefitTierStr := ""
	for i, tier := range c.VolumeBenefitTiers {
		if i > 1 {
			benefitTierStr += ", "
		}
		benefitTierStr += fmt.Sprintf("%d(minimumRunningNotionalTakerVolume(%s), volumeDiscountFactors(%s))",
			i,
			tier.MinimumRunningNotionalTakerVolume.String(),
			tier.VolumeDiscountFactors.String(),
		)
	}

	return fmt.Sprintf(
		"endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s)",
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
	)
}

func NewVolumeDiscountProgramChangesFromProto(v *vegapb.VolumeDiscountProgramChanges) *VolumeDiscountProgramChanges {
	if v == nil {
		return &VolumeDiscountProgramChanges{}
	}

	benefitTiers := make([]*VolumeBenefitTier, 0, len(v.BenefitTiers))
	for _, tier := range v.BenefitTiers {
		minimumRunningVolume, _ := num.UintFromString(tier.MinimumRunningNotionalTakerVolume, 10)

		benefitTiers = append(benefitTiers, &VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: minimumRunningVolume,
			VolumeDiscountFactors:             FactorsFromDiscountFactorsWithDefault(tier.VolumeDiscountFactors, tier.VolumeDiscountFactor),
		})
	}

	return &VolumeDiscountProgramChanges{
		EndOfProgramTimestamp: time.Unix(v.EndOfProgramTimestamp, 0),
		WindowLength:          v.WindowLength,
		VolumeBenefitTiers:    benefitTiers,
	}
}
