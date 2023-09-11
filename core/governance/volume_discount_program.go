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
		if tier.VolumeDiscountFactor.GreaterThan(maxDiscountFactor) {
			return types.ProposalErrorInvalidVolumeDiscountProgram, fmt.Errorf("tier %d defines a volume discount factor higher than the maximum allowed by the network parameter %q: maximum is %s, but got %s", i+1, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, maxDiscountFactor.String(), tier.VolumeDiscountFactor.String())
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
