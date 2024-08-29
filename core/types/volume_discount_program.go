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
	"code.vegaprotocol.io/vega/libs/ptr"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type VolumeDiscountStats struct {
	DiscountFactors Factors
}

type VolumeDiscountProgram struct {
	ID                    string
	Version               uint64
	EndOfProgramTimestamp time.Time
	WindowLength          uint64
	VolumeBenefitTiers    []*VolumeBenefitTier
}

type VolumeBenefitTier struct {
	MinimumRunningNotionalTakerVolume *num.Uint
	VolumeDiscountFactors             Factors
}

func (v VolumeDiscountProgram) IntoProto() *vegapb.VolumeDiscountProgram {
	benefitTiers := make([]*vegapb.VolumeBenefitTier, 0, len(v.VolumeBenefitTiers))
	for i, tier := range v.VolumeBenefitTiers {
		benefitTiers = append(benefitTiers, &vegapb.VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.String(),
			VolumeDiscountFactors:             tier.VolumeDiscountFactors.IntoDiscountFactorsProto(),
			TierNumber:                        ptr.From(uint64(i + 1)),
		})
	}

	return &vegapb.VolumeDiscountProgram{
		Version:               v.Version,
		Id:                    v.ID,
		BenefitTiers:          benefitTiers,
		EndOfProgramTimestamp: v.EndOfProgramTimestamp.Unix(),
		WindowLength:          v.WindowLength,
	}
}

func (v VolumeDiscountProgram) DeepClone() *VolumeDiscountProgram {
	benefitTiers := make([]*VolumeBenefitTier, 0, len(v.VolumeBenefitTiers))
	for _, tier := range v.VolumeBenefitTiers {
		benefitTiers = append(benefitTiers, &VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.Clone(),
			VolumeDiscountFactors:             tier.VolumeDiscountFactors.Clone(),
		})
	}

	cpy := VolumeDiscountProgram{
		ID:                    v.ID,
		Version:               v.Version,
		EndOfProgramTimestamp: v.EndOfProgramTimestamp,
		WindowLength:          v.WindowLength,
		VolumeBenefitTiers:    benefitTiers,
	}
	return &cpy
}

func NewVolumeDiscountProgramFromProto(v *vegapb.VolumeDiscountProgram) *VolumeDiscountProgram {
	if v == nil {
		return &VolumeDiscountProgram{}
	}

	benefitTiers := make([]*VolumeBenefitTier, 0, len(v.BenefitTiers))
	for _, tier := range v.BenefitTiers {
		minimumRunningVolume, _ := num.UintFromString(tier.MinimumRunningNotionalTakerVolume, 10)

		benefitTiers = append(benefitTiers, &VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: minimumRunningVolume,
			VolumeDiscountFactors:             FactorsFromDiscountFactorsWithDefault(tier.VolumeDiscountFactors, tier.VolumeDiscountFactor),
		})
	}

	return &VolumeDiscountProgram{
		ID:                    v.Id,
		Version:               v.Version,
		EndOfProgramTimestamp: time.Unix(v.EndOfProgramTimestamp, 0),
		WindowLength:          v.WindowLength,
		VolumeBenefitTiers:    benefitTiers,
	}
}

func (c VolumeDiscountProgram) String() string {
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
		"ID(%s), version(%d) endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s)",
		c.ID,
		c.Version,
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
	)
}
