package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminPurgePermissionsParams struct {
	Wallet string `json:"wallet"`
}
type AdminPurgePermissions struct {
	walletStore WalletStore
}

// Handle purges all the permissions set for all hostname.
func (h *AdminPurgePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validatePurgePermissionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	alreadyUnlocked, err := h.walletStore.IsWalletAlreadyUnlocked(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not verify whether the wallet is already unlock or not: %w", err))
	}
	if !alreadyUnlocked {
		return nil, requestNotPermittedError(ErrWalletIsLocked)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	w.PurgePermissions()

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
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

	return params, nil
}

func NewAdminPurgePermissions(
	walletStore WalletStore,
) *AdminPurgePermissions {
	return &AdminPurgePermissions{
		walletStore: walletStore,
	}
}
