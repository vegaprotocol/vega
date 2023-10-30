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

package governance

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
)

func validateUpdateReferralProgram(netp NetParams, p *types.UpdateReferralProgram, enactment int64) (types.ProposalError, error) {
	if enact := time.Unix(enactment, 0); enact.After(p.Changes.EndOfProgramTimestamp) {
		return types.ProposalErrorInvalidReferralProgram, fmt.Errorf("the proposal must be enacted before the referral program ends")
	}
	maxReferralTiers, _ := netp.GetUint(netparams.ReferralProgramMaxReferralTiers)
	if len(p.Changes.BenefitTiers) > int(maxReferralTiers.Uint64()) {
		return types.ProposalErrorInvalidReferralProgram, fmt.Errorf("the number of benefit tiers in the proposal is higher than the maximum allowed by the network parameter %q: maximum is %s, but got %d", netparams.ReferralProgramMaxReferralTiers, maxReferralTiers.String(), len(p.Changes.BenefitTiers))
	}

	if len(p.Changes.StakingTiers) > int(maxReferralTiers.Uint64()) {
		return types.ProposalErrorInvalidReferralProgram, fmt.Errorf("the number of staking tiers in the proposal is higher than the maximum allowed by the network parameter %q: maximum is %s, but got %d", netparams.ReferralProgramMaxReferralTiers, maxReferralTiers.String(), len(p.Changes.StakingTiers))
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
	return types.ProposalErrorUnspecified, nil
}

func updatedReferralProgramFromProposal(p *proposal) *types.ReferralProgram {
	terms := p.Terms.GetUpdateReferralProgram()

	return &types.ReferralProgram{
		ID:                    p.ID,
		EndOfProgramTimestamp: terms.Changes.EndOfProgramTimestamp,
		WindowLength:          terms.Changes.WindowLength,
		BenefitTiers:          terms.Changes.BenefitTiers,
		StakingTiers:          terms.Changes.StakingTiers,
	}
}
