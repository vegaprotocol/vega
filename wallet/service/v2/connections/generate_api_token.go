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
		return "", fmt.Errorf("could not verify the wallet existence: %w", err)
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
