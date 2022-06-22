// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
