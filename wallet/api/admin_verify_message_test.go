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
	t.Run("verify message with invalid params fails", testVerifyMessageWithInvalidParamsFails)
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
				PubKey: "",
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty message",
			params: api.AdminVerifyMessageParams{
				PubKey:         vgrand.RandomStr(5),
				EncodedMessage: "",
			},
			expectedError: api.ErrMessageIsRequired,
		},
		{
			name: "with non-base64 message",
			params: api.AdminVerifyMessageParams{
				PubKey:           vgrand.RandomStr(5),
				EncodedMessage:   "blahh",
				EncodedSignature: "sigsig",
			},
			expectedError: api.ErrEncodedMessageIsNotValidBase64String,
		},
		{
			name: "with empty signature",
			params: api.AdminVerifyMessageParams{
				PubKey:           vgrand.RandomStr(5),
				EncodedMessage:   "blah",
				EncodedSignature: "",
			},
			expectedError: api.ErrSignatureIsRequired,
		},
		{
			name: "with non-base64 signature",
			params: api.AdminVerifyMessageParams{
				PubKey:           vgrand.RandomStr(5),
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

func (h *verifyMessageHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminSignMessageResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
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
