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

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
)

func validateUpdateVolumeRebateProgram(netp NetParams, p *types.UpdateVolumeRebateProgram) (types.ProposalError, error) {
	maxTiers, _ := netp.GetUint(netparams.VolumeRebateProgramMaxBenefitTiers)
	if len(p.Changes.VolumeRebateBenefitTiers) > int(maxTiers.Uint64()) {
		return types.ProposalErrorInvalidVolumeRebateProgram, fmt.Errorf("the number of tiers in the proposal is higher than the maximum allowed by the network parameter %q: maximum is %s, but got %d", netparams.VolumeRebateProgramMaxBenefitTiers, maxTiers.String(), len(p.Changes.VolumeRebateBenefitTiers))
	}

	treasuryFee, _ := netp.GetDecimal(netparams.MarketFeeFactorsTreasuryFee)
	buybackFee, _ := netp.GetDecimal(netparams.MarketFeeFactorsBuyBackFee)
	maxRebate := treasuryFee.Add(buybackFee)

	for i, tier := range p.Changes.VolumeRebateBenefitTiers {
		if tier.AdditionalMakerRebate.GreaterThan(maxRebate) {
			return types.ProposalErrorInvalidVolumeRebateProgram, fmt.Errorf("tier %d defines an additional rebate factor higher than the maximum allowed by the network parameters: maximum is (%s+%s), but got %s", i+1, buybackFee.String(), treasuryFee.String(), tier.AdditionalMakerRebate.String())
		}
	}
	return 0, nil
}

func updatedVolumeRebateProgramFromProposal(p *proposal) *types.VolumeRebateProgram {
	terms := p.Terms.GetUpdateVolumeRebateProgram()

	return &types.VolumeRebateProgram{
		ID:                       p.ID,
		Version:                  terms.Changes.Version,
		EndOfProgramTimestamp:    terms.Changes.EndOfProgramTimestamp,
		WindowLength:             terms.Changes.WindowLength,
		VolumeRebateBenefitTiers: terms.Changes.VolumeRebateBenefitTiers,
	}
}
