package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminUpdatePermissionsParams struct {
	Wallet      string             `json:"wallet"`
	Hostname    string             `json:"hostname"`
	Permissions wallet.Permissions `json:"permissions"`
}

type AdminUpdatePermissionsResult struct {
	Permissions wallet.Permissions `json:"permissions"`
}

type AdminUpdatePermissions struct {
	walletStore WalletStore
}

func (h *AdminUpdatePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateUpdatePermissionsParams(rawParams)
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

	if err := w.UpdatePermissions(params.Hostname, params.Permissions); err != nil {
		return nil, InvalidParams(fmt.Errorf("could not update the permissions: %w", err))
	}

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AdminUpdatePermissionsResult{
		Permissions: w.Permissions(params.Hostname),
	}, nil
}

func validateUpdatePermissionsParams(rawParams jsonrpc.Params) (AdminUpdatePermissionsParams, error) {
	if rawParams == nil {
		return AdminUpdatePermissionsParams{}, ErrParamsRequired
	}

	params := AdminUpdatePermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminUpdatePermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminUpdatePermissionsParams{}, ErrWalletIsRequired
	}

	if params.Hostname == "" {
		return AdminUpdatePermissionsParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewAdminUpdatePermissions(
	walletStore WalletStore,
) *AdminUpdatePermissions {
	return &AdminUpdatePermissions{
		walletStore: walletStore,
	}
}
