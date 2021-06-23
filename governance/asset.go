package governance

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types"
	"github.com/pkg/errors"
)

var (
	ErrMissingERC20ContractAddress = errors.New("missing erc20 contract address")
	ErrMissingBuiltinAssetField    = errors.New("missing builtin asset field")
	ErrInvalidAssetDetails         = errors.New("invalid asset details")
)

func validateNewAsset(ad *types.AssetDetails) (proto.ProposalError, error) {
	if perr, err := validateCommonAssetDetails(ad); err != nil {
		return perr, err
	}

	switch s := ad.Source.(type) {
	case *types.AssetDetails_BuiltinAsset:
		return validateBuiltinAssetSource(s.BuiltinAsset)
	case *types.AssetDetails_Erc20:
		return validateERC20AssetSource(s.Erc20)
	default:
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, errors.New("unsupported asset source")
	}
}

func validateCommonAssetDetails(ad *types.AssetDetails) (proto.ProposalError, error) {
	if len(ad.Name) <= 0 || len(ad.Symbol) <= 0 || ad.Decimals == 0 || ad.TotalSupply.LTEUint64(0) || ad.MinLpStake.LTEUint64(0) {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			ErrInvalidAssetDetails
	}

	if ad.TotalSupply.IsZero() {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			ErrInvalidAssetDetails
	}

	if ad.MinLpStake.IsZero() {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			ErrInvalidAssetDetails
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateBuiltinAssetSource(ba *types.BuiltinAsset) (proto.ProposalError, error) {
	if ba.MaxFaucetAmountMint.IsZero() {
		return proto.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD, ErrMissingBuiltinAssetField
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateERC20AssetSource(ba *types.ERC20) (proto.ProposalError, error) {
	if len(ba.ContractAddress) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS, ErrMissingERC20ContractAddress
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
