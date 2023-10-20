// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type myAssetResolver VegaResolverRoot

func listAssetAccounts(ctx context.Context, client TradingDataServiceClientV2, asset *types.Asset, accountType types.AccountType) (*v2.AccountBalance, error) {
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

	var acc *v2.AccountBalance
	if len(res.Accounts.Edges) > 0 {
		acc = res.Accounts.Edges[0].Node
	}

	return acc, nil
}

func (r *myAssetResolver) InfrastructureFeeAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE)
}

func (r *myAssetResolver) GlobalRewardPoolAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD)
}

func (r *myAssetResolver) TakerFeeRewardAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES)
}

func (r *myAssetResolver) MakerFeeRewardAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES)
}

func (r *myAssetResolver) LpFeeRewardAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES)
}

func (r *myAssetResolver) MarketProposerRewardAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)
}

func (r *myAssetResolver) NetworkTreasuryAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY)
}

func (r *myAssetResolver) GlobalInsuranceAccount(ctx context.Context, asset *types.Asset) (*v2.AccountBalance, error) {
	return listAssetAccounts(ctx, r.tradingDataClientV2, asset, types.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE)
}

func (r myAssetResolver) Name(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.Name, nil
}

func (r myAssetResolver) Symbol(ctx context.Context, obj *types.Asset) (string, error) {
	return obj.Details.Symbol, nil
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
