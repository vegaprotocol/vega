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

func (r myAssetResolver) Name(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.Name, nil
}

func (r myAssetResolver) Symbol(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.Symbol, nil
}

func (r myAssetResolver) TotalSupply(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.TotalSupply, nil
}

func (r *myAssetResolver) Decimals(ctx context.Context, obj *types.Asset) (int, error) {
	return int(obj.Details.Decimals), nil
}

func (r *myAssetResolver) MinLpStake(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.MinLpStake, nil
}

func (r *myAssetResolver) Source(ctx context.Context, obj *types.Asset) (AssetSource, error) {
	return AssetSourceFromProto(obj.Details)
}

func AssetSourceFromProto(pdetails *types.AssetDetails) (AssetSource, error) {
	if pdetails == nil {
		return nil, ErrNilAssetSource
	}
	switch asimpl := pdetails.Source.(type) {
	case *types.AssetDetails_BuiltinAsset:
		return BuiltinAssetFromProto(asimpl.BuiltinAsset), nil
	case *types.AssetDetails_Erc20:
		return ERC20FromProto(asimpl.Erc20), nil
	default:
		return nil, ErrUnimplementedAssetSource
	}
}

func BuiltinAssetFromProto(ba *types.BuiltinAsset) *BuiltinAsset {
	return &BuiltinAsset{
		MaxFaucetAmountMint: ba.MaxFaucetAmountMint,
	}
}

func ERC20FromProto(ea *types.ERC20) *Erc20 {
	return &Erc20{
		ContractAddress: ea.ContractAddress,
	}
}
