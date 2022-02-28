package governance

import (
	"code.vegaprotocol.io/vega/types"
)

func validateNewAsset(ad *types.AssetDetails) (types.ProposalError, error) {
	if perr, err := validateCommonAssetDetails(ad); err != nil {
		return perr, err
	}
	if ad.Source == nil {
		return types.ProposalErrorUnspecified, nil
	}
	return ad.Source.ValidateAssetSource()
}

func validateCommonAssetDetails(ad *types.AssetDetails) (types.ProposalError, error) {
	if len(ad.Name) <= 0 {
		return types.ProposalErrorInvalidAssetDetails,
			types.ErrInvalidAssetNameEmpty
	}

	if len(ad.Symbol) <= 0 {
		return types.ProposalErrorInvalidAssetDetails,
			types.ErrInvalidAssetSymbolEmpty
	}

	if ad.Decimals == 0 {
		return types.ProposalErrorInvalidAssetDetails,
			types.ErrInvalidAssetDecimalPlacesZero
	}

	if ad.TotalSupply.IsZero() {
		return types.ProposalErrorInvalidAssetDetails,
			types.ErrInvalidAssetTotalSupplyZero
	}

	if ad.Quantum.IsZero() {
		return types.ProposalErrorInvalidAssetDetails,
			types.ErrInvalidAssetQuantumZero
	}

	return types.ProposalErrorUnspecified, nil
}
