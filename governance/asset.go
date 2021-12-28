package governance

import (
	proto "code.vegaprotocol.io/protos/vega"
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
	if len(ad.Name) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			types.ErrInvalidAssetNameEmpty
	}

	if len(ad.Symbol) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			types.ErrInvalidAssetSymbolEmpty
	}

	if ad.Decimals == 0 {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			types.ErrInvalidAssetDecimalPlacesZero
	}

	if ad.TotalSupply.IsZero() {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			types.ErrInvalidAssetTotalSupplyZero
	}

	if ad.MinLpStake.IsZero() {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			types.ErrInvalidAssetMinLPStakeZero
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
