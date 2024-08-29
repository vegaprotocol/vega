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

func validateUpdateVolumeDiscountProgram(netp NetParams, p *types.UpdateVolumeDiscountProgram) (types.ProposalError, error) {
	maxTiers, _ := netp.GetUint(netparams.VolumeDiscountProgramMaxBenefitTiers)
	if len(p.Changes.VolumeBenefitTiers) > int(maxTiers.Uint64()) {
		return types.ProposalErrorInvalidVolumeDiscountProgram, fmt.Errorf("the number of tiers in the proposal is higher than the maximum allowed by the network parameter %q: maximum is %s, but got %d", netparams.VolumeDiscountProgramMaxBenefitTiers, maxTiers.String(), len(p.Changes.VolumeBenefitTiers))
	}

	maxDiscountFactor, _ := netp.GetDecimal(netparams.VolumeDiscountProgramMaxVolumeDiscountFactor)
	for i, tier := range p.Changes.VolumeBenefitTiers {
		if tier.VolumeDiscountFactors.Infra.GreaterThan(maxDiscountFactor) {
			return types.ProposalErrorInvalidVolumeDiscountProgram, fmt.Errorf("tier %d defines a volume discount infrastructure factor higher than the maximum allowed by the network parameter %q: maximum is %s, but got %s", i+1, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, maxDiscountFactor.String(), tier.VolumeDiscountFactors.Infra.String())
		}
		if tier.VolumeDiscountFactors.Maker.GreaterThan(maxDiscountFactor) {
			return types.ProposalErrorInvalidVolumeDiscountProgram, fmt.Errorf("tier %d defines a volume discount maker factor higher than the maximum allowed by the network parameter %q: maximum is %s, but got %s", i+1, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, maxDiscountFactor.String(), tier.VolumeDiscountFactors.Maker.String())
		}
		if tier.VolumeDiscountFactors.Liquidity.GreaterThan(maxDiscountFactor) {
			return types.ProposalErrorInvalidVolumeDiscountProgram, fmt.Errorf("tier %d defines a volume discount liquidity factor higher than the maximum allowed by the network parameter %q: maximum is %s, but got %s", i+1, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, maxDiscountFactor.String(), tier.VolumeDiscountFactors.Liquidity.String())
		}
	}
	return 0, nil
}

func updatedVolumeDiscountProgramFromProposal(p *proposal) *types.VolumeDiscountProgram {
	terms := p.Terms.GetUpdateVolumeDiscountProgram()

	return &types.VolumeDiscountProgram{
		ID:                    p.ID,
		Version:               terms.Changes.Version,
		EndOfProgramTimestamp: terms.Changes.EndOfProgramTimestamp,
		WindowLength:          terms.Changes.WindowLength,
		VolumeBenefitTiers:    terms.Changes.VolumeBenefitTiers,
	}
}
