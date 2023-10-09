// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	PublicKey        string `json:"publicKey"`
	EncodedMessage   string `json:"encodedMessage"`
	EncodedSignature string `json:"encodedSignature"`
}

type AdminVerifyMessageResult struct {
	IsValid bool `json:"isValid"`
}

type AdminVerifyMessage struct{}

func (h *AdminVerifyMessage) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminVerifyMessageParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	m, err := base64.StdEncoding.DecodeString(params.EncodedMessage)
	if err != nil {
		return nil, InvalidParams(ErrEncodedMessageIsNotValidBase64String)
	}

	s, err := base64.StdEncoding.DecodeString(params.EncodedSignature)
	if err != nil {
		return nil, InvalidParams(ErrEncodedSignatureIsNotValidBase64String)
	}

	decodedPubKey, err := hex.DecodeString(params.PublicKey)
	if err != nil {
		return nil, InvalidParams(fmt.Errorf("could not decode the public key: %w", err))
	}

	signatureAlgorithm := crypto.NewEd25519()
	valid, err := signatureAlgorithm.Verify(decodedPubKey, m, s)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the signature: %w", err))
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

	if params.PublicKey == "" {
		return AdminVerifyMessageParams{}, ErrPublicKeyIsRequired
	}

	if params.EncodedMessage == "" {
		return AdminVerifyMessageParams{}, ErrMessageIsRequired
	}

	if params.EncodedSignature == "" {
		return AdminVerifyMessageParams{}, ErrSignatureIsRequired
	}

	return AdminVerifyMessageParams{
		PublicKey:        params.PublicKey,
		EncodedMessage:   params.EncodedMessage,
		EncodedSignature: params.EncodedSignature,
	}, nil
}
