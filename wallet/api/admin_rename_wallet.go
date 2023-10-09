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
	"github.com/mitchellh/mapstructure"
)

type AdminRenameWalletParams struct {
	Wallet  string `json:"wallet"`
	NewName string `json:"newName"`
}

type AdminRenameWallet struct {
	walletStore WalletStore
}

// Handle renames a wallet.
func (h *AdminRenameWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRenameWalletParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, InvalidParams(ErrWalletDoesNotExist)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.NewName); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if exist {
		return nil, InvalidParams(ErrWalletAlreadyExists)
	}

	if err := h.walletStore.RenameWallet(ctx, params.Wallet, params.NewName); err != nil {
		return nil, InternalError(fmt.Errorf("could not rename the wallet: %w", err))
	}

	return nil, nil
}

func validateRenameWalletParams(rawParams jsonrpc.Params) (AdminRenameWalletParams, error) {
	if rawParams == nil {
		return AdminRenameWalletParams{}, ErrParamsRequired
	}

	params := AdminRenameWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRenameWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminRenameWalletParams{}, ErrWalletIsRequired
	}

	if params.NewName == "" {
		return AdminRenameWalletParams{}, ErrNewNameIsRequired
	}

	return params, nil
}

func NewAdminRenameWallet(
	walletStore WalletStore,
) *AdminRenameWallet {
	return &AdminRenameWallet{
		walletStore: walletStore,
	}
}
