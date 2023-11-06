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

type AdminCreateWalletParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type AdminCreateWalletResult struct {
	Wallet AdminCreatedWallet  `json:"wallet"`
	Key    AdminFirstPublicKey `json:"key"`
}

type AdminCreatedWallet struct {
	Name                 string `json:"name"`
	KeyDerivationVersion uint32 `json:"keyDerivationVersion"`
	RecoveryPhrase       string `json:"recoveryPhrase"`
}

type AdminFirstPublicKey struct {
	PublicKey string            `json:"publicKey"`
	Algorithm wallet.Algorithm  `json:"algorithm"`
	Metadata  []wallet.Metadata `json:"metadata"`
}

type AdminCreateWallet struct {
	walletStore WalletStore
}

// Handle creates a wallet and generates its first key.
func (h *AdminCreateWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateCreateWalletParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if exist {
		return nil, InvalidParams(ErrWalletAlreadyExists)
	}

	w, recoveryPhrase, err := wallet.NewHDWallet(params.Wallet)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not create the HD wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not generate the first key: %w", err))
	}

	if err := h.walletStore.CreateWallet(ctx, w, params.Passphrase); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AdminCreateWalletResult{
		Wallet: AdminCreatedWallet{
			Name:                 w.Name(),
			KeyDerivationVersion: w.KeyDerivationVersion(),
			RecoveryPhrase:       recoveryPhrase,
		},
		Key: AdminFirstPublicKey{
			PublicKey: kp.PublicKey(),
			Algorithm: wallet.Algorithm{
				Name:    kp.AlgorithmName(),
				Version: kp.AlgorithmVersion(),
			},
			Metadata: kp.Metadata(),
		},
	}, nil
}

func validateCreateWalletParams(rawParams jsonrpc.Params) (AdminCreateWalletParams, error) {
	if rawParams == nil {
		return AdminCreateWalletParams{}, ErrParamsRequired
	}

	params := AdminCreateWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCreateWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminCreateWalletParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminCreateWalletParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewAdminCreateWallet(
	walletStore WalletStore,
) *AdminCreateWallet {
	return &AdminCreateWallet{
		walletStore: walletStore,
	}
}
