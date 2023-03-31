package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminRevokePermissionsParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
	Hostname   string `json:"hostname"`
}
type AdminRevokePermissions struct {
	walletStore WalletStore
}

// Handle revokes the permissions set on the specified hostname.
func (h *AdminRevokePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRevokePermissionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if err := h.walletStore.UnlockWallet(ctx, params.Wallet, params.Passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, invalidParams(err)
		}
		return nil, internalError(fmt.Errorf("could not unlock the wallet: %w", err))
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	w.RevokePermissions(params.Hostname)

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return nil, nil
}

func validateRevokePermissionsParams(rawParams jsonrpc.Params) (AdminRevokePermissionsParams, error) {
	if rawParams == nil {
		return AdminRevokePermissionsParams{}, ErrParamsRequired
	}

	params := AdminRevokePermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRevokePermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminRevokePermissionsParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminRevokePermissionsParams{}, ErrPassphraseIsRequired
	}

	if params.Hostname == "" {
		return AdminRevokePermissionsParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewAdminRevokePermissions(
	walletStore WalletStore,
) *AdminRevokePermissions {
	return &AdminRevokePermissions{
		walletStore: walletStore,
	}
}
