package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type DescribeKeyParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
	PublicKey  string `json:"publicKey"`
}

type DescribeKeyResult struct {
	PublicKey string            `json:"publicKey"`
	Algorithm wallet.Algorithm  `json:"algorithm"`
	Metadata  []wallet.Metadata `json:"metadata"`
	IsTainted bool              `json:"isTainted"`
}

type DescribeKey struct {
	walletStore WalletStore
}

// Handle retrieve key's information.
func (h *DescribeKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribeKeyParams(rawParams)
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
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	if !w.HasPublicKey(params.PublicKey) {
		return nil, invalidParams(ErrPublicKeyDoesNotExist)
	}

	publicKey, err := w.DescribePublicKey(params.PublicKey)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the key: %w", err))
	}

	return DescribeKeyResult{
		PublicKey: publicKey.Key(),
		Algorithm: wallet.Algorithm{
			Name:    publicKey.AlgorithmName(),
			Version: publicKey.AlgorithmVersion(),
		},
		Metadata:  publicKey.Metadata(),
		IsTainted: publicKey.IsTainted(),
	}, nil
}

func validateDescribeKeyParams(rawParams jsonrpc.Params) (DescribeKeyParams, error) {
	if rawParams == nil {
		return DescribeKeyParams{}, ErrParamsRequired
	}

	params := DescribeKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return DescribeKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return DescribeKeyParams{}, ErrWalletIsRequired
	}

	if params.PublicKey == "" {
		return DescribeKeyParams{}, ErrPublicKeyIsRequired
	}

	if params.Passphrase == "" {
		return DescribeKeyParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewDescribeKey(
	walletStore WalletStore,
) *DescribeKey {
	return &DescribeKey{
		walletStore: walletStore,
	}
}
