package governance

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types"
)

func validateNewAsset(ad *types.AssetDetails) (types.ProposalError, error) {
	if perr, err := validateCommonAssetDetails(ad); err != nil {
		return perr, err
	}
	if ad.Source == nil {
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
	}
	return ad.Source.ValidateAssetSource()
}

func validateCommonAssetDetails(ad *types.AssetDetails) (proto.ProposalError, error) {
	if len(ad.Name) <= 0 || len(ad.Symbol) <= 0 || ad.Decimals == 0 || ad.TotalSupply.IsZero() || ad.MinLpStake.IsZero() {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			types.ErrInvalidAssetDetails
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
