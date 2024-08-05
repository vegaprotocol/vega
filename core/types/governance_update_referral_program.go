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

type ProposalTermsUpdateReferralProgram struct {
	UpdateReferralProgram *UpdateReferralProgram
}

func (a ProposalTermsUpdateReferralProgram) String() string {
	return fmt.Sprintf(
		"updateReferralProgram(%s)",
		stringer.PtrToString(a.UpdateReferralProgram),
	)
}

func (a ProposalTermsUpdateReferralProgram) isPTerm() {}

func (a ProposalTermsUpdateReferralProgram) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateReferralProgram{
		UpdateReferralProgram: a.UpdateReferralProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateReferralProgram) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateReferralProgram{
		UpdateReferralProgram: a.UpdateReferralProgram.IntoProto(),
	}
}

func (a ProposalTermsUpdateReferralProgram) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateReferralProgram
}

func (a ProposalTermsUpdateReferralProgram) DeepClone() ProposalTerm {
	if a.UpdateReferralProgram == nil {
		return &ProposalTermsUpdateReferralProgram{}
	}
	return &ProposalTermsUpdateReferralProgram{
		UpdateReferralProgram: a.UpdateReferralProgram.DeepClone(),
	}
}

func NewUpdateReferralProgramProposalFromProto(
	updateReferralProgramProto *vegapb.UpdateReferralProgram,
) (*ProposalTermsUpdateReferralProgram, error) {
	return &ProposalTermsUpdateReferralProgram{
		UpdateReferralProgram: NewUpdateReferralProgramFromProto(updateReferralProgramProto),
	}, nil
}

type UpdateReferralProgram struct {
	Changes *ReferralProgramChanges
}

func (p UpdateReferralProgram) IntoProto() *vegapb.UpdateReferralProgram {
	return &vegapb.UpdateReferralProgram{
		Changes: p.Changes.IntoProto(),
	}
}

func (p UpdateReferralProgram) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.PtrToString(p.Changes),
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
		Changes: NewReferralProgramChangesFromProto(p.Changes),
	}
}

type ReferralProgramChanges struct {
	EndOfProgramTimestamp time.Time
	WindowLength          uint64
	BenefitTiers          []*BenefitTier
	StakingTiers          []*StakingTier
}

func (c ReferralProgramChanges) IntoProto() *vegapb.ReferralProgramChanges {
	benefitTiers := make([]*vegapb.BenefitTier, 0, len(c.BenefitTiers))
	for _, tier := range c.BenefitTiers {
		benefitTiers = append(benefitTiers, &vegapb.BenefitTier{
			MinimumEpochs:                     tier.MinimumEpochs.String(),
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.String(),
			ReferralRewardFactors:             tier.ReferralRewardFactors.IntoRewardFactorsProto(),
			ReferralDiscountFactors:           tier.ReferralDiscountFactors.IntoDiscountFactorsProto(),
		})
	}

	stakingTiers := make([]*vegapb.StakingTier, 0, len(c.StakingTiers))
	for _, tier := range c.StakingTiers {
		stakingTiers = append(stakingTiers, &vegapb.StakingTier{
			MinimumStakedTokens:      tier.MinimumStakedTokens.String(),
			ReferralRewardMultiplier: tier.ReferralRewardMultiplier.String(),
		})
	}

	return &vegapb.ReferralProgramChanges{
		BenefitTiers:          benefitTiers,
		StakingTiers:          stakingTiers,
		EndOfProgramTimestamp: c.EndOfProgramTimestamp.Unix(),
		WindowLength:          c.WindowLength,
	}
}

func (c ReferralProgramChanges) DeepClone() *ReferralProgramChanges {
	benefitTiers := make([]*BenefitTier, 0, len(c.BenefitTiers))
	for _, tier := range c.BenefitTiers {
		benefitTiers = append(benefitTiers, &BenefitTier{
			MinimumEpochs:                     tier.MinimumEpochs.Clone(),
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.Clone(),
			ReferralRewardFactors:             tier.ReferralRewardFactors.Clone(),
			ReferralDiscountFactors:           tier.ReferralDiscountFactors.Clone(),
		})
	}

	stakingTiers := make([]*StakingTier, 0, len(c.StakingTiers))
	for _, tier := range c.StakingTiers {
		stakingTiers = append(stakingTiers, &StakingTier{
			MinimumStakedTokens:      tier.MinimumStakedTokens.Clone(),
			ReferralRewardMultiplier: tier.ReferralRewardMultiplier,
		})
	}

	cpy := ReferralProgramChanges{
		EndOfProgramTimestamp: c.EndOfProgramTimestamp,
		WindowLength:          c.WindowLength,
		BenefitTiers:          benefitTiers,
		StakingTiers:          stakingTiers,
	}
	return &cpy
}

func (c ReferralProgramChanges) String() string {
	benefitTierStr := ""
	for i, tier := range c.BenefitTiers {
		if i > 1 {
			benefitTierStr += ", "
		}
		benefitTierStr += fmt.Sprintf("%d(minimumEpochs(%s), minimumRunningNotionalTakerVolume(%s), referralRewardFactors(%s), referralDiscountFactors(%s))",
			i,
			tier.MinimumEpochs.String(),
			tier.MinimumRunningNotionalTakerVolume.String(),
			tier.ReferralRewardFactors.String(),
			tier.ReferralDiscountFactors.String(),
		)
	}

	stakingTierStr := ""
	for i, tier := range c.StakingTiers {
		if i > 1 {
			stakingTierStr += ", "
		}
		stakingTierStr += fmt.Sprintf("%d(minimumStakedTokens(%s), referralRewardMultiplier(%s))",
			i,
			tier.MinimumStakedTokens.String(),
			tier.ReferralRewardMultiplier.String(),
		)
	}

	return fmt.Sprintf(
		"endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s), stakingTiers(%s)",
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
		stakingTierStr,
	)
}

func NewReferralProgramChangesFromProto(c *vegapb.ReferralProgramChanges) *ReferralProgramChanges {
	if c == nil {
		return nil
	}

	benefitTiers := make([]*BenefitTier, 0, len(c.BenefitTiers))
	for _, tier := range c.BenefitTiers {
		minimumEpochs, _ := num.UintFromString(tier.MinimumEpochs, 10)
		minimumRunningVolume, _ := num.UintFromString(tier.MinimumRunningNotionalTakerVolume, 10)

		benefitTiers = append(benefitTiers, &BenefitTier{
			MinimumEpochs:                     minimumEpochs,
			MinimumRunningNotionalTakerVolume: minimumRunningVolume,
			ReferralRewardFactors:             FactorsFromRewardFactorsWithDefault(tier.ReferralRewardFactors, tier.ReferralRewardFactor),
			ReferralDiscountFactors:           FactorsFromDiscountFactorsWithDefault(tier.ReferralDiscountFactors, tier.ReferralDiscountFactor),
		})
	}

	stakingTiers := make([]*StakingTier, 0, len(c.StakingTiers))
	for _, tier := range c.StakingTiers {
		minimumStakedTokens, _ := num.UintFromString(tier.MinimumStakedTokens, 10)
		referralRewardMultiplier, _ := num.DecimalFromString(tier.ReferralRewardMultiplier)

		stakingTiers = append(stakingTiers, &StakingTier{
			MinimumStakedTokens:      minimumStakedTokens,
			ReferralRewardMultiplier: referralRewardMultiplier,
		})
	}

	return &ReferralProgramChanges{
		EndOfProgramTimestamp: time.Unix(c.EndOfProgramTimestamp, 0),
		WindowLength:          c.WindowLength,
		BenefitTiers:          benefitTiers,
		StakingTiers:          stakingTiers,
	}
}
