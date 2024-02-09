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

package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAdminVerifyMessage(t *testing.T) {
	t.Run("Documentation matches the code", testAdminVerifyMessageSchemaCorrect)
	t.Run("verify message with invalid params fails", testVerifyMessageWithInvalidParamsFails)
}

func testAdminVerifyMessageSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.verify_message", api.AdminVerifyMessageParams{}, api.AdminVerifyMessageResult{})
}

func testVerifyMessageWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		},
		{
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		},
		{
			name: "with empty publickey",
			params: api.AdminVerifyMessageParams{
				PublicKey: "",
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty message",
			params: api.AdminVerifyMessageParams{
				PublicKey:      vgrand.RandomStr(5),
				EncodedMessage: "",
			},
			expectedError: api.ErrMessageIsRequired,
		},
		{
			name: "with non-base64 message",
			params: api.AdminVerifyMessageParams{
				PublicKey:        vgrand.RandomStr(5),
				EncodedMessage:   "blahh",
				EncodedSignature: "sigsig",
			},
			expectedError: api.ErrEncodedMessageIsNotValidBase64String,
		},
		{
			name: "with empty signature",
			params: api.AdminVerifyMessageParams{
				PublicKey:        vgrand.RandomStr(5),
				EncodedMessage:   "blah",
				EncodedSignature: "",
			},
			expectedError: api.ErrSignatureIsRequired,
		},
		{
			name: "with non-base64 signature",
			params: api.AdminVerifyMessageParams{
				PublicKey:        vgrand.RandomStr(5),
				EncodedMessage:   "blah",
				EncodedSignature: "blahh",
			},
			expectedError: api.ErrEncodedSignatureIsNotValidBase64String,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newVerifyMessageHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

type verifyMessageHandler struct {
	*api.AdminVerifyMessage
	ctrl *gomock.Controller
}

func (h *verifyMessageHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminSignMessageResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminSignMessageResult)
		if !ok {
			t.Fatal("AdminUpdatePermissions handler result is not a AdminSignTransactionResult")
		}
		return result, err
	}
	return api.AdminSignMessageResult{}, err
}

func newVerifyMessageHandler(t *testing.T) *verifyMessageHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	return &verifyMessageHandler{
		AdminVerifyMessage: api.NewAdminVerifyMessage(),
		ctrl:               ctrl,
	}
}
