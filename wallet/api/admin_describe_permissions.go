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

package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"

	"github.com/mitchellh/mapstructure"
)

type AdminDescribePermissionsParams struct {
	Wallet   string `json:"wallet"`
	Hostname string `json:"hostname"`
}

type AdminDescribePermissionsResult struct {
	Permissions wallet.Permissions `json:"permissions"`
}

type AdminDescribePermissions struct {
	walletStore WalletStore
}

// Handle retrieves permissions set for the specified wallet and hostname.
func (h *AdminDescribePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribePermissionsParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, InvalidParams(ErrWalletDoesNotExist)
	}

	alreadyUnlocked, err := h.walletStore.IsWalletAlreadyUnlocked(ctx, params.Wallet)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not verify whether the wallet is already unlock or not: %w", err))
	}
	if !alreadyUnlocked {
		return nil, RequestNotPermittedError(ErrWalletIsLocked)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	return AdminDescribePermissionsResult{
		Permissions: w.Permissions(params.Hostname),
	}, nil
}

func validateDescribePermissionsParams(rawParams jsonrpc.Params) (AdminDescribePermissionsParams, error) {
	if rawParams == nil {
		return AdminDescribePermissionsParams{}, ErrParamsRequired
	}

	params := AdminDescribePermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribePermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminDescribePermissionsParams{}, ErrWalletIsRequired
	}

	if params.Hostname == "" {
		return AdminDescribePermissionsParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewAdminDescribePermissions(
	walletStore WalletStore,
) *AdminDescribePermissions {
	return &AdminDescribePermissions{
		walletStore: walletStore,
	}
}
