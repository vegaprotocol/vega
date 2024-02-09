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

type AdminTaintKeyParams struct {
	Wallet    string `json:"wallet"`
	PublicKey string `json:"publicKey"`
}

type AdminTaintKey struct {
	walletStore WalletStore
}

// Handle marks the specified public key as tainted. It makes it unusable for
// transaction signing and sending.
func (h *AdminTaintKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateTaintKeyParams(rawParams)
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

	if !w.HasPublicKey(params.PublicKey) {
		return nil, InvalidParams(ErrPublicKeyDoesNotExist)
	}

	if err := w.TaintKey(params.PublicKey); err != nil {
		return nil, InternalError(fmt.Errorf("could not taint the key: %w", err))
	}

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return nil, nil
}

func validateTaintKeyParams(rawParams jsonrpc.Params) (AdminTaintKeyParams, error) {
	if rawParams == nil {
		return AdminTaintKeyParams{}, ErrParamsRequired
	}

	params := AdminTaintKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminTaintKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminTaintKeyParams{}, ErrWalletIsRequired
	}

	if params.PublicKey == "" {
		return AdminTaintKeyParams{}, ErrPublicKeyIsRequired
	}

	return params, nil
}

func NewAdminTaintKey(
	walletStore WalletStore,
) *AdminTaintKey {
	return &AdminTaintKey{
		walletStore: walletStore,
	}
}
