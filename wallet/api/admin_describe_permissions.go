package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminDescribePermissionsParams struct {
	Wallet   string `json:"wallet"`
	Hostname string `json:"hostname"`
}

type AdminDescribePermissionsResult struct {
	Permissions wallet.Permissions `json:"permissions"`
}

type AdminDescribePermissions struct {
	walletStore WalletStore
}

// Handle retrieves permissions set for the specified wallet and hostname.
func (h *AdminDescribePermissions) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribePermissionsParams(rawParams)
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

	return AdminDescribePermissionsResult{
		Permissions: w.Permissions(params.Hostname),
	}, nil
}

func validateDescribePermissionsParams(rawParams jsonrpc.Params) (AdminDescribePermissionsParams, error) {
	if rawParams == nil {
		return AdminDescribePermissionsParams{}, ErrParamsRequired
	}

	params := AdminDescribePermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribePermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminDescribePermissionsParams{}, ErrWalletIsRequired
	}

	if params.Hostname == "" {
		return AdminDescribePermissionsParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewAdminDescribePermissions(
	walletStore WalletStore,
) *AdminDescribePermissions {
	return &AdminDescribePermissions{
		walletStore: walletStore,
	}
}
