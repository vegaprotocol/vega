package api

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/crypto"
	"github.com/mitchellh/mapstructure"
)

type AdminVerifyMessageParams struct {
	PubKey           string `json:"pubKey"`
	EncodedMessage   string `json:"encodedMessage"`
	EncodedSignature string `json:"encodedSignature"`
}

type AdminVerifyMessageResult struct {
	IsValid bool `json:"valid"`
}

type AdminVerifyMessage struct{}

func (h *AdminVerifyMessage) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminVerifyMessageParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	m, err := base64.StdEncoding.DecodeString(params.EncodedMessage)
	if err != nil {
		return nil, invalidParams(ErrEncodedMessageIsNotValidBase64String)
	}

	s, err := base64.StdEncoding.DecodeString(params.EncodedSignature)
	if err != nil {
		return nil, invalidParams(ErrEncodedSignatureIsNotValidBase64String)
	}

	decodedPubKey, err := hex.DecodeString(params.PubKey)
	if err != nil {
		return nil, invalidParams(fmt.Errorf("could not decode the public key: %w", err))
	}

	signatureAlgorithm := crypto.NewEd25519()
	valid, err := signatureAlgorithm.Verify(decodedPubKey, m, s)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not verify the signature: %w", err))
	}

	return AdminVerifyMessageResult{
		IsValid: valid,
	}, nil
}

func NewAdminVerifyMessage() *AdminVerifyMessage {
	return &AdminVerifyMessage{}
}

func validateAdminVerifyMessageParams(rawParams jsonrpc.Params) (AdminVerifyMessageParams, error) {
	if rawParams == nil {
		return AdminVerifyMessageParams{}, ErrParamsRequired
	}

	params := AdminVerifyMessageParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminVerifyMessageParams{}, ErrParamsDoNotMatch
	}

	if params.PubKey == "" {
		return AdminVerifyMessageParams{}, ErrPublicKeyIsRequired
	}

	if params.EncodedMessage == "" {
		return AdminVerifyMessageParams{}, ErrMessageIsRequired
	}

	if params.EncodedSignature == "" {
		return AdminVerifyMessageParams{}, ErrSignatureIsRequired
	}

	return AdminVerifyMessageParams{
		PubKey:           params.PubKey,
		EncodedMessage:   params.EncodedMessage,
		EncodedSignature: params.EncodedSignature,
	}, nil
}
