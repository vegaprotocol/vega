package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminRevokePermissionsParams struct {
	Wallet   string `json:"wallet"`
	Hostname string `json:"hostname"`
}
type AdminRevokePermissions struct {
	walletStore WalletStore
}

// Handle revokes the permissions set on the specified hostname.
func (h *AdminRevokePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRevokePermissionsParams(rawParams)
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

	w.RevokePermissions(params.Hostname)

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the wallet: %w", err))
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
