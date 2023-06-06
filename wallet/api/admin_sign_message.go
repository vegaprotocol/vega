package api

import (
	"context"
	"encoding/base64"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminSignMessageParams struct {
	Wallet         string `json:"wallet"`
	PublicKey      string `json:"publicKey"`
	EncodedMessage string `json:"encodedMessage"`
}

type AdminSignMessageResult struct {
	EncodedSignature string `json:"encodedSignature"`
}

type AdminSignMessage struct {
	walletStore WalletStore
}

func (h *AdminSignMessage) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminSignMessageParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	m, err := base64.StdEncoding.DecodeString(params.EncodedMessage)
	if err != nil {
		return nil, InvalidParams(ErrEncodedMessageIsNotValidBase64String)
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

	signature, err := w.SignAny(params.PublicKey, m)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not sign the message: %w", err))
	}

	return AdminSignMessageResult{
		EncodedSignature: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

func NewAdminSignMessage(walletStore WalletStore) *AdminSignMessage {
	return &AdminSignMessage{
		walletStore: walletStore,
	}
}

func validateAdminSignMessageParams(rawParams jsonrpc.Params) (AdminSignMessageParams, error) {
	if rawParams == nil {
		return AdminSignMessageParams{}, ErrParamsRequired
	}

	params := AdminSignMessageParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminSignMessageParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminSignMessageParams{}, ErrWalletIsRequired
	}

	if params.PublicKey == "" {
		return AdminSignMessageParams{}, ErrPublicKeyIsRequired
	}

	if params.EncodedMessage == "" {
		return AdminSignMessageParams{}, ErrMessageIsRequired
	}

	return AdminSignMessageParams{
		Wallet:         params.Wallet,
		PublicKey:      params.PublicKey,
		EncodedMessage: params.EncodedMessage,
	}, nil
}
