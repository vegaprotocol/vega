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
)

type AdminListWalletsResult struct {
	Wallets []string `json:"wallets"`
}

type AdminListWallets struct {
	walletStore WalletStore
}

// Handle list all the wallets present on the computer.
func (h *AdminListWallets) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	wallets, err := h.walletStore.ListWallets(ctx)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not list the wallets: %w", err))
	}

	return AdminListWalletsResult{
		Wallets: wallets,
	}, nil
}

func NewAdminListWallets(
	walletStore WalletStore,
) *AdminListWallets {
	return &AdminListWallets{
		walletStore: walletStore,
	}
}
