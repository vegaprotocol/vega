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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type VolumeRebateStats struct {
	RebateFactor num.Decimal
}

type VolumeRebateProgram struct {
	ID                       string
	Version                  uint64
	EndOfProgramTimestamp    time.Time
	WindowLength             uint64
	VolumeRebateBenefitTiers []*VolumeRebateBenefitTier
}

type VolumeRebateBenefitTier struct {
	MinimumPartyMakerVolumeFraction num.Decimal
	AdditionalMakerRebate           num.Decimal
}

func (v VolumeRebateProgram) IntoProto() *vegapb.VolumeRebateProgram {
	benefitTiers := make([]*vegapb.VolumeRebateBenefitTier, 0, len(v.VolumeRebateBenefitTiers))
	for _, tier := range v.VolumeRebateBenefitTiers {
		benefitTiers = append(benefitTiers, &vegapb.VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: tier.MinimumPartyMakerVolumeFraction.String(),
			AdditionalMakerRebate:           tier.AdditionalMakerRebate.String(),
		})
	}

	return &vegapb.VolumeRebateProgram{
		Version:               v.Version,
		Id:                    v.ID,
		BenefitTiers:          benefitTiers,
		EndOfProgramTimestamp: v.EndOfProgramTimestamp.Unix(),
		WindowLength:          v.WindowLength,
	}
}

func (v VolumeRebateProgram) DeepClone() *VolumeRebateProgram {
	benefitTiers := make([]*VolumeRebateBenefitTier, 0, len(v.VolumeRebateBenefitTiers))
	for _, tier := range v.VolumeRebateBenefitTiers {
		benefitTiers = append(benefitTiers, &VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: tier.MinimumPartyMakerVolumeFraction,
			AdditionalMakerRebate:           tier.AdditionalMakerRebate,
		})
	}

	cpy := VolumeRebateProgram{
		ID:                       v.ID,
		Version:                  v.Version,
		EndOfProgramTimestamp:    v.EndOfProgramTimestamp,
		WindowLength:             v.WindowLength,
		VolumeRebateBenefitTiers: benefitTiers,
	}
	return &cpy
}

func NewVolumeRebateProgramFromProto(v *vegapb.VolumeRebateProgram) *VolumeRebateProgram {
	if v == nil {
		return &VolumeRebateProgram{}
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

	return &VolumeRebateProgram{
		ID:                       v.Id,
		Version:                  v.Version,
		EndOfProgramTimestamp:    time.Unix(v.EndOfProgramTimestamp, 0),
		WindowLength:             v.WindowLength,
		VolumeRebateBenefitTiers: benefitTiers,
	}
}

func (c VolumeRebateProgram) String() string {
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
		"ID(%s), version(%d) endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s)",
		c.ID,
		c.Version,
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
	)
}
