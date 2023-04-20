package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAdminSignMessage(t *testing.T) {
	t.Run("Documentation matches the code", testAdminSignMessageSchemaCorrect)
	t.Run("Signing message with invalid params fails", testSigningMessageWithInvalidParamsFails)
	t.Run("Signing message with wallet that doesn't exist fails", testSigningMessageWithWalletThatDoesntExistFails)
	t.Run("Signing message failing to get wallet fails", testSigningMessageFailingToGetWalletFails)
}

func testAdminSignMessageSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.sign_message", api.AdminSignMessageParams{}, api.AdminSignMessageResult{})
}

func testSigningMessageWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty wallet",
			params: api.AdminSignMessageParams{
				Wallet: "",
			},
			expectedError: api.ErrWalletIsRequired,
		},
		{
			name: "with empty public key",
			params: api.AdminSignMessageParams{
				Wallet: vgrand.RandomStr(5),
				PubKey: "",
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
		{
			name: "with empty message",
			params: api.AdminSignMessageParams{
				Wallet:         vgrand.RandomStr(5),
				PubKey:         vgrand.RandomStr(5),
				EncodedMessage: "",
			},
			expectedError: api.ErrMessageIsRequired,
		},
		{
			name: "with non-base64 message",
			params: api.AdminSignMessageParams{
				Wallet:         vgrand.RandomStr(5),
				PubKey:         vgrand.RandomStr(5),
				EncodedMessage: "blahh",
			},
			expectedError: api.ErrEncodedMessageIsNotValidBase64String,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newSignMessageHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

func testSigningMessageWithWalletThatDoesntExistFails(t *testing.T) {
	// given
	ctx := context.Background()
	params := paramsWithMessage("bXltZXNzYWdl")

	// setup
	handler := newSignMessageHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, params.Wallet).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, params)

	// then
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
	assert.Empty(t, result)
}

func testSigningMessageFailingToGetWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	params := paramsWithMessage("bXltZXNzYWdl")

	// setup
	handler := newSignMessageHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, params.Wallet).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, params.Wallet).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, params.Wallet).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, params)

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
	assert.Empty(t, result)
}

func paramsWithMessage(m string) api.AdminSignMessageParams {
	return api.AdminSignMessageParams{
		Wallet:         vgrand.RandomStr(5),
		PubKey:         vgrand.RandomStr(5),
		EncodedMessage: m,
	}
}

type signMessageHandler struct {
	*api.AdminSignMessage
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *signMessageHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminSignMessageResult, *jsonrpc.ErrorDetails) {
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

func newSignMessageHandler(t *testing.T) *signMessageHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &signMessageHandler{
		AdminSignMessage: api.NewAdminSignMessage(walletStore),
		ctrl:             ctrl,
		walletStore:      walletStore,
	}
}
