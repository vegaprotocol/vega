package api

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"github.com/mitchellh/mapstructure"
)

type AdminGenerateAPITokenParams struct {
	Description string                            `json:"name"`
	Expiry      *int64                            `json:"expiry"`
	Wallet      AdminGenerateAPITokenWalletParams `json:"wallet"`
}

type AdminGenerateAPITokenWalletParams struct {
	Name       string `json:"name"`
	Passphrase string `json:"passphrase"`
}

type AdminGenerateAPITokenResult struct {
	Token string `json:"token"`
}

type AdminGenerateAPIToken struct {
	walletStore WalletStore
	tokenStore  TokenStore
}

// Handle generates a long-living API token.
func (h *AdminGenerateAPIToken) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminGenerateAPITokenParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet.Name); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if _, err := h.walletStore.GetWallet(ctx, params.Wallet.Name, params.Wallet.Passphrase); err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	token := session.Token{
		Description: params.Description,
		Token:       session.GenerateToken(),
		Expiry:      params.Expiry,
		Wallet: session.WalletCredentials{
			Name:       params.Wallet.Name,
			Passphrase: params.Wallet.Passphrase,
		},
	}

	if err := h.tokenStore.SaveToken(token); err != nil {
		return nil, internalError(fmt.Errorf("could not save the newly generated token: %w", err))
	}

	return AdminGenerateAPITokenResult{
		Token: token.Token,
	}, nil
}

func validateAdminGenerateAPITokenParams(rawParams jsonrpc.Params) (AdminGenerateAPITokenParams, error) {
	if rawParams == nil {
		return AdminGenerateAPITokenParams{}, ErrParamsRequired
	}

	params := AdminGenerateAPITokenParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminGenerateAPITokenParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet.Name == "" {
		return AdminGenerateAPITokenParams{}, ErrWalletNameIsRequired
	}

	if params.Wallet.Passphrase == "" {
		return AdminGenerateAPITokenParams{}, ErrWalletPassphraseIsRequired
	}

	if params.Expiry != nil {
		if time.Now().After(time.Unix(*params.Expiry, 0)) {
			return AdminGenerateAPITokenParams{}, ErrAPITokenExpiryInThePast
		}
	}

	return params, nil
}

func NewAdminGenerateAPIToken(
	walletStore WalletStore,
	tokenStore TokenStore,
) *AdminGenerateAPIToken {
	return &AdminGenerateAPIToken{
		walletStore: walletStore,
		tokenStore:  tokenStore,
	}
}
