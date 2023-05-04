package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminDescribeWalletParams struct {
	Wallet string `json:"wallet"`
}

type AdminDescribeWalletResult struct {
	Name                 string `json:"name"`
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	KeyDerivationVersion uint32 `json:"keyDerivationVersion"`
}

type AdminDescribeWallet struct {
	walletStore WalletStore
}

func (h *AdminDescribeWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribeWalletParams(rawParams)
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

	return AdminDescribeWalletResult{
		Name:                 w.Name(),
		ID:                   w.ID(),
		Type:                 w.Type(),
		KeyDerivationVersion: w.KeyDerivationVersion(),
	}, nil
}

func validateDescribeWalletParams(rawParams jsonrpc.Params) (AdminDescribeWalletParams, error) {
	if rawParams == nil {
		return AdminDescribeWalletParams{}, ErrParamsRequired
	}

	params := AdminDescribeWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribeWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminDescribeWalletParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminDescribeWallet(
	walletStore WalletStore,
) *AdminDescribeWallet {
	return &AdminDescribeWallet{
		walletStore: walletStore,
	}
}
