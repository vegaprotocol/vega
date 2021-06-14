package governance

import (
	"strconv"

	"code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrMissingERC20ContractAddress = errors.New("missing erc20 contract address")
	ErrMissingBuiltinAssetField    = errors.New("missing builtin asset field")
)

func validateNewAsset(as *proto.AssetSource) (proto.ProposalError, error) {
	switch s := as.Source.(type) {
	case *proto.AssetSource_BuiltinAsset:
		return validateBuiltinAssetSource(s.BuiltinAsset)
	case *proto.AssetSource_Erc20:
		return validateERC20AssetSource(s.Erc20)
	default:
		return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, errors.New("unsupported asset source")
	}
}

func validateBuiltinAssetSource(ba *proto.BuiltinAsset) (proto.ProposalError, error) {
	if len(ba.Name) <= 0 || len(ba.Symbol) <= 0 || ba.Decimals == 0 || len(ba.TotalSupply) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD, ErrMissingBuiltinAssetField
	}

	u, err := strconv.ParseUint(ba.TotalSupply, 10, 64)
	if err != nil || u == 0 {
		return proto.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD, ErrMissingBuiltinAssetField
	}
	u, err = strconv.ParseUint(ba.MaxFaucetAmountMint, 10, 64)
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
