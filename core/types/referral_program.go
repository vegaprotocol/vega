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
}

type BenefitTier struct {
	MinimumEpochs                     *num.Uint
	MinimumRunningNotionalTakerVolume *num.Uint
	ReferralRewardFactor              num.Decimal
	ReferralDiscountFactor            num.Decimal
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
			tier.ReferralRewardFactor.String(),
			tier.ReferralDiscountFactor.String(),
		)
	}

	return fmt.Sprintf(
		"endOfProgramTimestamp(%d), windowLength(%d), benefitTiers(%s)",
		c.EndOfProgramTimestamp.Unix(),
		c.WindowLength,
		benefitTierStr,
	)
}

func (c ReferralProgram) IntoProto() *vegapb.ReferralProgram {
	benefitTiers := make([]*vegapb.BenefitTier, 0, len(c.BenefitTiers))
	for _, tier := range c.BenefitTiers {
		benefitTiers = append(benefitTiers, &vegapb.BenefitTier{
			MinimumEpochs:                     tier.MinimumEpochs.String(),
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.String(),
			ReferralRewardFactor:              tier.ReferralRewardFactor.String(),
			ReferralDiscountFactor:            tier.ReferralDiscountFactor.String(),
		})
	}

	return &vegapb.ReferralProgram{
		Version:               c.Version,
		Id:                    c.ID,
		BenefitTiers:          benefitTiers,
		EndOfProgramTimestamp: c.EndOfProgramTimestamp.Unix(),
		WindowLength:          c.WindowLength,
	}
}

func (c ReferralProgram) DeepClone() *ReferralProgram {
	benefitTiers := make([]*BenefitTier, 0, len(c.BenefitTiers))
	for _, tier := range c.BenefitTiers {
		benefitTiers = append(benefitTiers, &BenefitTier{
			MinimumEpochs:                     tier.MinimumEpochs.Clone(),
			MinimumRunningNotionalTakerVolume: tier.MinimumRunningNotionalTakerVolume.Clone(),
			ReferralRewardFactor:              tier.ReferralRewardFactor,
			ReferralDiscountFactor:            tier.ReferralDiscountFactor,
		})
	}

	cpy := ReferralProgram{
		ID:                    c.ID,
		Version:               c.Version,
		EndOfProgramTimestamp: c.EndOfProgramTimestamp,
		WindowLength:          c.WindowLength,
		BenefitTiers:          benefitTiers,
	}
	return &cpy
}

func NewReferralProgramFromProto(c *vegapb.ReferralProgram) *ReferralProgram {
	if c == nil {
		return &ReferralProgram{}
	}

	benefitTiers := make([]*BenefitTier, 0, len(c.BenefitTiers))
	for _, tier := range c.BenefitTiers {
		minimumEpochs, _ := num.UintFromString(tier.MinimumEpochs, 10)
		minimumRunningVolume, _ := num.UintFromString(tier.MinimumRunningNotionalTakerVolume, 10)
		rewardFactor, _ := num.DecimalFromString(tier.ReferralRewardFactor)
		discountFactor, _ := num.DecimalFromString(tier.ReferralDiscountFactor)

		benefitTiers = append(benefitTiers, &BenefitTier{
			MinimumEpochs:                     minimumEpochs,
			MinimumRunningNotionalTakerVolume: minimumRunningVolume,
			ReferralRewardFactor:              rewardFactor,
			ReferralDiscountFactor:            discountFactor,
		})
	}

	return &ReferralProgram{
		ID:                    c.Id,
		Version:               c.Version,
		EndOfProgramTimestamp: time.Unix(0, c.EndOfProgramTimestamp),
		WindowLength:          c.WindowLength,
		BenefitTiers:          benefitTiers,
	}
}
