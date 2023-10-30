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

package connections

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/wallet/api"
)

type GenerateAPITokenParams struct {
	Description string                       `json:"name"`
	ExpiresIn   *time.Duration               `json:"expireIn"`
	Wallet      GenerateAPITokenWalletParams `json:"wallet"`
}

type GenerateAPITokenWalletParams struct {
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

type GenerateAPITokenHandler struct {
	walletStore api.WalletStore
	tokenStore  TokenStore
	timeService TimeService
}

func (h *GenerateAPITokenHandler) Handle(ctx context.Context, params GenerateAPITokenParams) (Token, error) {
	if params.ExpiresIn != nil && *params.ExpiresIn == 0 {
		return "", ErrExpirationDurationMustBeGreaterThan0
	}

	if params.Wallet.Name == "" {
		return "", ErrWalletNameIsRequired
	}

	if params.Wallet.Passphrase == "" {
		return "", ErrWalletPassphraseIsRequired
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet.Name); err != nil {
		return "", fmt.Errorf("could not verify the wallet exists: %w", err)
	} else if !exist {
		return "", api.ErrWalletDoesNotExist
	}

	if err := h.walletStore.UnlockWallet(ctx, params.Wallet.Name, params.Wallet.Passphrase); err != nil {
		return "", fmt.Errorf("could not unlock the wallet: %w", err)
	}

	if _, err := h.walletStore.GetWallet(ctx, params.Wallet.Name); err != nil {
		return "", fmt.Errorf("could not retrieve the wallet: %w", err)
	}

	now := h.timeService.Now().Truncate(time.Second)

	var expirationDate *time.Time
	if params.ExpiresIn != nil {
		ed := now.Add(*params.ExpiresIn).Truncate(time.Second)
		expirationDate = &ed
	}

	tokenDescription := TokenDescription{
		Description:    params.Description,
		Token:          GenerateToken(),
		CreationDate:   now,
		ExpirationDate: expirationDate,
		Wallet: WalletCredentials{
			Name:       params.Wallet.Name,
			Passphrase: params.Wallet.Passphrase,
		},
	}

	if err := h.tokenStore.SaveToken(tokenDescription); err != nil {
		return "", fmt.Errorf("could not save the newly generated token: %w", err)
	}

	return tokenDescription.Token, nil
}

func NewGenerateAPITokenHandler(
	walletStore api.WalletStore,
	tokenStore TokenStore,
	timeService TimeService,
) *GenerateAPITokenHandler {
	return &GenerateAPITokenHandler{
		walletStore: walletStore,
		tokenStore:  tokenStore,
		timeService: timeService,
	}
}
