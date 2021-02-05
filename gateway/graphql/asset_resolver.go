package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
)

type myAssetResolver VegaResolverRoot

func (r *myAssetResolver) InfrastructureFeeAccount(ctx context.Context, obj *types.Asset) (*types.Account, error) {
	if len(obj.Id) <= 0 {
		return nil, ErrMissingIDOrReference
	}
	req := &protoapi.FeeInfrastructureAccountsRequest{
		Asset: obj.Id,
	}
	res, err := r.tradingDataClient.FeeInfrastructureAccounts(ctx, req)
	if err != nil {
		return nil, err
	}

	var acc *types.Account
	if len(res.Accounts) > 0 {
		acc = res.Accounts[0]
	}

	return acc, nil
}

func (r *myAssetResolver) Decimals(ctx context.Context, obj *types.Asset) (int, error) {
	return int(obj.Decimals), nil
}

func (r *myAssetResolver) Source(ctx context.Context, obj *types.Asset) (AssetSource, error) {
	return AssetSourceFromProto(obj.Source)
}

func AssetSourceFromProto(psource *types.AssetSource) (AssetSource, error) {
	if psource == nil {
		return nil, ErrNilAssetSource
	}
	switch asimpl := psource.Source.(type) {
	case *types.AssetSource_BuiltinAsset:
		return BuiltinAssetFromProto(asimpl.BuiltinAsset), nil
	case *types.AssetSource_Erc20:
		return ERC20FromProto(asimpl.Erc20), nil
	default:
		return nil, ErrUnimplementedAssetSource
	}
}

func BuiltinAssetFromProto(ba *types.BuiltinAsset) *BuiltinAsset {
	return &BuiltinAsset{
		Name:                ba.Name,
		Symbol:              ba.Symbol,
		TotalSupply:         ba.TotalSupply,
		Decimals:            int(ba.Decimals),
		MaxFaucetAmountMint: ba.MaxFaucetAmountMint,
	}
}

func ERC20FromProto(ea *types.ERC20) *Erc20 {
	return &Erc20{
		ContractAddress: ea.ContractAddress,
	}
}
