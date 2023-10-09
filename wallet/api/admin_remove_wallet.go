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

type AdminRemoveWalletParams struct {
	Wallet string `json:"wallet"`
}

type AdminRemoveWallet struct {
	walletStore WalletStore
}

// Handle removes a wallet from the computer.
func (h *AdminRemoveWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRemoveWalletParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, InvalidParams(ErrWalletDoesNotExist)
	}

	if err := h.walletStore.DeleteWallet(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not remove the wallet: %w", err))
	}

	return nil, nil
}

func validateRemoveWalletParams(rawParams jsonrpc.Params) (AdminRemoveWalletParams, error) {
	if rawParams == nil {
		return AdminRemoveWalletParams{}, ErrParamsRequired
	}

	params := AdminRemoveWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRemoveWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminRemoveWalletParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminRemoveWallet(
	walletStore WalletStore,
) *AdminRemoveWallet {
	return &AdminRemoveWallet{
		walletStore: walletStore,
	}
}
