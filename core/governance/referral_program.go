package governance

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
)

func validateUpdateReferralProgram(netp NetParams, p *types.UpdateReferralProgram) (types.ProposalError, error) {
	maxBenefitTiers, _ := netp.GetUint(netparams.ReferralProgramMaxBenefitTiers)
	if len(p.Changes.BenefitTiers) > int(maxBenefitTiers.Uint64()) {
		return types.ProposalErrorInvalidReferralProgram, fmt.Errorf("the number of tiers in the proposal is higher than the maximum allowed by the network parameter %q: maximum is %s, but got %d", netparams.ReferralProgramMaxBenefitTiers, maxBenefitTiers.String(), len(p.Changes.BenefitTiers))
	}

	maxRewardFactor, _ := netp.GetDecimal(netparams.ReferralProgramMaxReferralRewardFactor)
	maxDiscountFactor, _ := netp.GetDecimal(netparams.ReferralProgramMaxReferralDiscountFactor)
	for i, tier := range p.Changes.BenefitTiers {
		if tier.ReferralRewardFactor.GreaterThan(maxRewardFactor) {
			return types.ProposalErrorInvalidReferralProgram, fmt.Errorf("tier %d defines a referral reward factor higher than the maximum allowed by the network parameter %q: maximum is %s, but got %s", i+1, netparams.ReferralProgramMaxReferralRewardFactor, maxRewardFactor.String(), tier.ReferralRewardFactor.String())
		}
		if tier.ReferralDiscountFactor.GreaterThan(maxDiscountFactor) {
			return types.ProposalErrorInvalidReferralProgram, fmt.Errorf("tier %d defines a referral discount factor higher than the maximum allowed by the network parameter %q: maximum is %s, but got %s", i+1, netparams.ReferralProgramMaxReferralDiscountFactor, maxDiscountFactor.String(), tier.ReferralDiscountFactor.String())
		}
	}
	return 0, nil
}

func updatedReferralProgramFromProposal(p *proposal) *types.ReferralProgram {
	terms := p.Terms.GetUpdateReferralProgram()

	return &types.ReferralProgram{
		ID:                    p.ID,
		EndOfProgramTimestamp: terms.Changes.EndOfProgramTimestamp,
		WindowLength:          terms.Changes.WindowLength,
		BenefitTiers:          terms.Changes.BenefitTiers,
	}
}
