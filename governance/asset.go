package governance

import (
	"strconv"

	"code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrMissingERC20ContractAddress = errors.New("missing erc20 contract address")
	ErrMissingBuiltinAssetField    = errors.New("missing builtin asset field")
	ErrInvalidAssetDetails         = errors.New("invalid asset details")
)

func validateNewAsset(ad *proto.AssetDetails) (proto.ProposalError, error) {
	if perr, err := validateCommonAssetDetails(ad); err != nil {
		return perr, err
	}

	switch s := ad.Source.(type) {
	case *proto.AssetDetails_BuiltinAsset:
		return validateBuiltinAssetSource(s.BuiltinAsset)
	case *proto.AssetDetails_Erc20:
		return validateERC20AssetSource(s.Erc20)
	default:
		return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, errors.New("unsupported asset source")
	}
}

func validateCommonAssetDetails(ad *proto.AssetDetails) (proto.ProposalError, error) {
	if len(ad.Name) <= 0 || len(ad.Symbol) <= 0 || ad.Decimals == 0 || len(ad.TotalSupply) <= 0 || len(ad.MinLpStake) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			ErrInvalidAssetDetails
	}

	u, err := strconv.ParseUint(ad.TotalSupply, 10, 64)
	if err != nil || u == 0 {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			ErrInvalidAssetDetails
	}

	u, err = strconv.ParseUint(ad.MinLpStake, 10, 64)
	if err != nil || u == 0 {
		return proto.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS,
			ErrInvalidAssetDetails
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateBuiltinAssetSource(ba *proto.BuiltinAsset) (proto.ProposalError, error) {
	u, err := strconv.ParseUint(ba.MaxFaucetAmountMint, 10, 64)
	if err != nil || u == 0 {
		return proto.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD, ErrMissingBuiltinAssetField
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateERC20AssetSource(ba *proto.ERC20) (proto.ProposalError, error) {
	if len(ba.ContractAddress) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS, ErrMissingERC20ContractAddress
	}

	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
