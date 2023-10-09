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

type AdminListPermissionsParams struct {
	Wallet string `json:"wallet"`
}

type AdminListPermissionsResult struct {
	Permissions map[string]wallet.PermissionsSummary `json:"permissions"`
}

type AdminListPermissions struct {
	walletStore WalletStore
}

// Handle returns the permissions summary for all set hostnames.
func (h *AdminListPermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateListPermissionsParams(rawParams)
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

	permissions := map[string]wallet.PermissionsSummary{}
	for _, hostname := range w.PermittedHostnames() {
		permissions[hostname] = w.Permissions(hostname).Summary()
	}

	return AdminListPermissionsResult{
		Permissions: permissions,
	}, nil
}

func validateListPermissionsParams(rawParams jsonrpc.Params) (AdminListPermissionsParams, error) {
	if rawParams == nil {
		return AdminListPermissionsParams{}, ErrParamsRequired
	}

	params := AdminListPermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminListPermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminListPermissionsParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminListPermissions(
	walletStore WalletStore,
) *AdminListPermissions {
	return &AdminListPermissions{
		walletStore: walletStore,
	}
}
