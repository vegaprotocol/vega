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

type ReferralProgram struct {
	ID                    string
	Version               uint64
	EndOfProgramTimestamp time.Time
	WindowLength          uint64
	BenefitTiers          []*BenefitTier
	StakingTiers          []*StakingTier
}

type Factors struct {
	Infra     num.Decimal
	Liquidity num.Decimal
	Maker     num.Decimal
}

var EmptyFactors = Factors{
	Infra:     num.DecimalZero(),
	Liquidity: num.DecimalZero(),
	Maker:     num.DecimalZero(),
}

func (f Factors) String() string {
	return fmt.Sprintf("infra(%s),liquidity(%s),maker(%s)", f.Infra.String(), f.Liquidity.String(), f.Maker.String())
}

func (f Factors) Equal(other Factors) bool {
	return f.Infra.Equal(other.Infra) && f.Maker.Equal(other.Maker) && f.Liquidity.Equal(other.Liquidity)
}

func (f Factors) CapRewardFactors(multiplier, referralProgramMaxRewardProportion num.Decimal) Factors {
	return Factors{
		Infra:     num.MinD(f.Infra.Mul(multiplier), referralProgramMaxRewardProportion),
		Maker:     num.MinD(f.Maker.Mul(multiplier), referralProgramMaxRewardProportion),
		Liquidity: num.MinD(f.Liquidity.Mul(multiplier), referralProgramMaxRewardProportion),
	}
}

func (f Factors) Clone() Factors {
	return Factors{
		Infra:     f.Infra,
		Liquidity: f.Liquidity,
		Maker:     f.Maker,
	}
}

func FactorsFromRewardFactorsWithDefault(factors *vegapb.RewardFactors, defaultFactor string) Factors {
	f := Factors{}
	if len(defaultFactor) > 0 {
		defaultFactorDec := num.MustDecimalFromString(defaultFactor)
		f.Infra = defaultFactorDec
		f.Maker = defaultFactorDec
		f.Liquidity = defaultFactorDec
	}
	if factors != nil {
		f.Infra, _ = num.DecimalFromString(factors.InfrastructureRewardFactor)
		f.Maker, _ = num.DecimalFromString(factors.MakerRewardFactor)
		f.Liquidity, _ = num.DecimalFromString(factors.LiquidityRewardFactor)
	}
	return f
}

func FactorsFromDiscountFactorsWithDefault(factors *vegapb.DiscountFactors, defaultFactor string) Factors {
	f := Factors{}
	if len(defaultFactor) > 0 {
		defaultFactorDec := num.MustDecimalFromString(defaultFactor)
		f.Infra = defaultFactorDec
		f.Maker = defaultFactorDec
		f.Liquidity = defaultFactorDec
	}
	if factors != nil {
		f.Infra, _ = num.DecimalFromString(factors.InfrastructureDiscountFactor)
		f.Maker, _ = num.DecimalFromString(factors.MakerDiscountFactor)
		f.Liquidity, _ = num.DecimalFromString(factors.LiquidityDiscountFactor)
	}
	return f
}

func (f Factors) IntoRewardFactorsProto() *vegapb.RewardFactors {
	factors := &vegapb.RewardFactors{}
	factors.InfrastructureRewardFactor = f.Infra.String()
	factors.MakerRewardFactor = f.Maker.String()
	factors.LiquidityRewardFactor = f.Liquidity.String()
	return factors
}

func (f Factors) IntoDiscountFactorsProto() *vegapb.DiscountFactors {
	factors := &vegapb.DiscountFactors{}
	factors.InfrastructureDiscountFactor = f.Infra.String()
	factors.MakerDiscountFactor = f.Maker.String()
	factors.LiquidityDiscountFactor = f.Liquidity.String()
	return factors
}

type BenefitTier struct {
	MinimumEpochs                     *num.Uint
	MinimumRunningNotionalTakerVolume *num.Uint
	ReferralRewardFactors             Factors
	ReferralDiscountFactors           Factors
}

type StakingTier struct {
	MinimumStakedTokens      *num.Uint
	ReferralRewardMultiplier num.Decimal
}

func (c ReferralProgram) String() string {
	benefitTierStr := ""
	for i, tier := range c.BenefitTiers {
		if i > 1 {
			benefitTierStr += ", "
		}
		benefitTierStr += fmt.Sprintf("%d(minimumEpochs(%s), minimumRunningNotionalTakerVolume(%s), referralRewardFactor(%s), referralDiscountFactor(%s))",
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
		"ID(%s) version(%d) endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s), stakingTiers(%s)",
		c.ID,
		c.Version,
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
		stakingTierStr,
	)
}

func (c ReferralProgram) IntoProto() *vegapb.ReferralProgram {
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

	return &vegapb.ReferralProgram{
		Version:               c.Version,
		Id:                    c.ID,
		BenefitTiers:          benefitTiers,
		StakingTiers:          stakingTiers,
		EndOfProgramTimestamp: c.EndOfProgramTimestamp.Unix(),
		WindowLength:          c.WindowLength,
	}
}

func NewReferralProgramFromProto(c *vegapb.ReferralProgram) *ReferralProgram {
	if c == nil {
		return &ReferralProgram{}
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

	return &ReferralProgram{
		ID:                    c.Id,
		Version:               c.Version,
		EndOfProgramTimestamp: time.Unix(c.EndOfProgramTimestamp, 0),
		WindowLength:          c.WindowLength,
		BenefitTiers:          benefitTiers,
		StakingTiers:          stakingTiers,
	}
}
