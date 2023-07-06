package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminListPermissionsParams struct {
	Wallet string `json:"wallet"`
}

type AdminListPermissionsResult struct {
	Permissions map[string]wallet.PermissionsSummary `json:"permissions"`
}

type AdminListPermissions struct {
	walletStore WalletStore
}

// Handle returns the permissions summary for all set hostnames.
func (h *AdminListPermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateListPermissionsParams(rawParams)
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

	return params, nil
}

func NewAdminListPermissions(
	walletStore WalletStore,
) *AdminListPermissions {
	return &AdminListPermissions{
		walletStore: walletStore,
	}
}
