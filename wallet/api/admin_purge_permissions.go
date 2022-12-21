package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminPurgePermissionsParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}
type AdminPurgePermissions struct {
	walletStore WalletStore
}

// Handle purges all the permissions set for all hostname.
func (h *AdminPurgePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validatePurgePermissionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet, params.Passphrase)
	if err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, invalidParams(err)
		}
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	w.PurgePermissions()

	if err := h.walletStore.SaveWallet(ctx, w, params.Passphrase); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return nil, nil
}

func validatePurgePermissionsParams(rawParams jsonrpc.Params) (AdminPurgePermissionsParams, error) {
	if rawParams == nil {
		return AdminPurgePermissionsParams{}, ErrParamsRequired
	}

	params := AdminPurgePermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminPurgePermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminPurgePermissionsParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminPurgePermissionsParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewAdminPurgePermissions(
	walletStore WalletStore,
) *AdminPurgePermissions {
	return &AdminPurgePermissions{
		walletStore: walletStore,
	}
}
