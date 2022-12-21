package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminListPermissionsParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type AdminListPermissionsResult struct {
	Permissions map[string]wallet.PermissionsSummary `json:"permissions"`
}

type AdminListPermissions struct {
	walletStore WalletStore
}

// Handle returns the permissions summary for all set hostnames.
func (h *AdminListPermissions) Handle(ctx context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateListPermissionsParams(rawParams)
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

	permissions := map[string]wallet.PermissionsSummary{}
	for _, hostname := range w.PermittedHostnames() {
		permissions[hostname] = w.Permissions(hostname).Summary()
	}

	return AdminListPermissionsResult{
		Permissions: permissions,
	}, nil
}

func validateListPermissionsParams(rawParams jsonrpc.Params) (AdminListPermissionsParams, error) {
	if rawParams == nil {
		return AdminListPermissionsParams{}, ErrParamsRequired
	}

	params := AdminListPermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminListPermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminListPermissionsParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminListPermissionsParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewAdminListPermissions(
	walletStore WalletStore,
) *AdminListPermissions {
	return &AdminListPermissions{
		walletStore: walletStore,
	}
}
