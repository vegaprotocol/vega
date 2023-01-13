package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminDescribeKeyParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
	PublicKey  string `json:"publicKey"`
}

type AdminDescribeKeyResult struct {
	PublicKey string            `json:"publicKey"`
	Name      string            `json:"name"`
	Algorithm wallet.Algorithm  `json:"algorithm"`
	Metadata  []wallet.Metadata `json:"metadata"`
	IsTainted bool              `json:"isTainted"`
}

type AdminDescribeKey struct {
	walletStore WalletStore
}

// Handle retrieves key's information.
func (h *AdminDescribeKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribeKeyParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
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

	if !w.HasPublicKey(params.PublicKey) {
		return nil, invalidParams(ErrPublicKeyDoesNotExist)
	}

	publicKey, err := w.DescribePublicKey(params.PublicKey)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the key: %w", err))
	}

	return AdminDescribeKeyResult{
		PublicKey: publicKey.Key(),
		Name:      publicKey.Name(),
		Algorithm: wallet.Algorithm{
			Name:    publicKey.AlgorithmName(),
			Version: publicKey.AlgorithmVersion(),
		},
		Metadata:  publicKey.Metadata(),
		IsTainted: publicKey.IsTainted(),
	}, nil
}

func validateDescribeKeyParams(rawParams jsonrpc.Params) (AdminDescribeKeyParams, error) {
	if rawParams == nil {
		return AdminDescribeKeyParams{}, ErrParamsRequired
	}

	params := AdminDescribeKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribeKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminDescribeKeyParams{}, ErrWalletIsRequired
	}

	if params.PublicKey == "" {
		return AdminDescribeKeyParams{}, ErrPublicKeyIsRequired
	}

	if params.Passphrase == "" {
		return AdminDescribeKeyParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewAdminDescribeKey(
	walletStore WalletStore,
) *AdminDescribeKey {
	return &AdminDescribeKey{
		walletStore: walletStore,
	}
}
