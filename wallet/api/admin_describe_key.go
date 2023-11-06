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

type AdminDescribeKeyParams struct {
	Wallet    string `json:"wallet"`
	PublicKey string `json:"publicKey"`
}

type AdminDescribeKeyResult struct {
	PublicKey string            `json:"publicKey"`
	Name      string            `json:"name"`
	Algorithm wallet.Algorithm  `json:"algorithm"`
	Metadata  []wallet.Metadata `json:"metadata"`
	IsTainted bool              `json:"isTainted"`
}

type AdminDescribeKey struct {
	walletStore WalletStore
}

// Handle retrieves key's information.
func (h *AdminDescribeKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribeKeyParams(rawParams)
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

	publicKey, err := w.DescribePublicKey(params.PublicKey)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not retrieve the key: %w", err))
	}

	return AdminDescribeKeyResult{
		PublicKey: publicKey.Key(),
		Name:      publicKey.Name(),
		Algorithm: wallet.Algorithm{
			Name:    publicKey.AlgorithmName(),
			Version: publicKey.AlgorithmVersion(),
		},
		Metadata:  publicKey.Metadata(),
		IsTainted: publicKey.IsTainted(),
	}, nil
}

func validateDescribeKeyParams(rawParams jsonrpc.Params) (AdminDescribeKeyParams, error) {
	if rawParams == nil {
		return AdminDescribeKeyParams{}, ErrParamsRequired
	}

	params := AdminDescribeKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribeKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminDescribeKeyParams{}, ErrWalletIsRequired
	}

	if params.PublicKey == "" {
		return AdminDescribeKeyParams{}, ErrPublicKeyIsRequired
	}

	return params, nil
}

func NewAdminDescribeKey(
	walletStore WalletStore,
) *AdminDescribeKey {
	return &AdminDescribeKey{
		walletStore: walletStore,
	}
}
