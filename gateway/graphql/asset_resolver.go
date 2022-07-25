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

package gql

import (
	"context"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	types "code.vegaprotocol.io/protos/vega"
)

type myAssetResolver VegaResolverRoot

func (r *myAssetResolver) Status(ctx context.Context, obj *types.Asset) (AssetStatus, error) {
	return convertAssetStatusFromProto(obj.Status)
}

func listAssetAccounts(ctx context.Context, client TradingDataServiceClientV2, asset *types.Asset, accountType types.AccountType) (*types.Account, error) {
	if asset == nil || len(asset.Id) <= 0 {
		return nil, ErrMissingIDOrReference
	}

	req := &v2.ListAccountsRequest{
		Filter: &v2.AccountFilter{
			AssetId:      asset.Id,
			AccountTypes: []types.AccountType{accountType},
		},
	}

	res, err := client.ListAccounts(ctx, req)
	if err != nil {
		return nil, err
	}

	var acc *types.Account
	if len(res.Accounts.Edges) > 0 {
		acc = res.Accounts.Edges[0].Account
	}

	return acc, nil
}

func (r *myAssetResolver) InfrastructureFeeAccount(ctx context.Context, asset *types.Asset) (*types.Account, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE)
}

func (r *myAssetResolver) GlobalRewardPoolAccount(ctx context.Context, asset *types.Asset) (*types.Account, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD)
}

func (r *myAssetResolver) TakerFeeRewardAccount(ctx context.Context, asset *types.Asset) (*types.Account, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_TAKER_PAID_FEES)
}

func (r *myAssetResolver) MakerFeeRewardAccount(ctx context.Context, asset *types.Asset) (*types.Account, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES)
}

func (r *myAssetResolver) LpFeeRewardAccount(ctx context.Context, asset *types.Asset) (*types.Account, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES)
}

func (r *myAssetResolver) MarketProposerRewardAccount(ctx context.Context, asset *types.Asset) (*types.Account, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)
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

func (r *myAssetResolver) Quantum(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.Quantum, nil
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
		ContractAddress:   ea.ContractAddress,
		LifetimeLimit:     ea.LifetimeLimit,
		WithdrawThreshold: ea.WithdrawThreshold,
	}
}
