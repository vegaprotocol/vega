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
	"github.com/stretchr/testify/require"
)

func TestAdminTaintKey(t *testing.T) {
	t.Run("Tainting a key with invalid params fails", testTaintingKeyWithInvalidParamsFails)
	t.Run("Tainting a key with valid params succeeds", testTaintingKeyWithValidParamsSucceeds)
	t.Run("Tainting a key on unknown wallet fails", testTaintingKeyOnUnknownWalletFails)
	t.Run("Tainting a key on unknown key fails", testTaintingKeyOnUnknownKeyFails)
	t.Run("Getting internal error during wallet verification doesn't taint the key", testGettingInternalErrorDuringWalletVerificationDoesNotTaintKey)
	t.Run("Getting internal error during wallet retrieval doesn't taint the key", testGettingInternalErrorDuringWalletRetrievalDoesNotTaintKey)
	t.Run("Getting internal error during wallet saving doesn't taint the key", testGettingInternalErrorDuringWalletSavingDoesNotTaintKey)
}

func testTaintingKeyWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		}, {
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		}, {
			name: "with empty name",
			params: api.AdminTaintKeyParams{
				Wallet:     "",
				PublicKey:  "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminTaintKeyParams{
				PublicKey:  "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				Wallet:     vgrand.RandomStr(5),
				Passphrase: "",
			},
			expectedError: api.ErrPassphraseIsRequired,
		}, {
			name: "with empty public key",
			params: api.AdminTaintKeyParams{
				PublicKey:  "",
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newTaintKeyHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// the
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testTaintingKeyWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, kp := walletWithKey(t)

	// setup
	handler := newTaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, expectedWallet, passphrase).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminTaintKeyParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
		PublicKey:  kp.PublicKey(),
	})

	// then
	require.Nil(t, errorDetails)
	require.True(t, expectedWallet.ListKeyPairs()[0].IsTainted())
}

func testTaintingKeyOnUnknownWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newTaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminTaintKeyParams{
		Wallet:     name,
		PublicKey:  vgrand.RandomStr(5),
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testTaintingKeyOnUnknownKeyFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)

	// setup
	handler := newTaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminTaintKeyParams{
		Wallet:     expectedWallet.Name(),
		PublicKey:  vgrand.RandomStr(5),
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrPublicKeyDoesNotExist)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotTaintKey(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, kp := walletWithKey(t)

	// setup
	handler := newTaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminTaintKeyParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
		PublicKey:  kp.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotTaintKey(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, kp := walletWithKey(t)

	// setup
	handler := newTaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(nil, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminTaintKeyParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
		PublicKey:  kp.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletSavingDoesNotTaintKey(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, kp := walletWithKey(t)

	// setup
	handler := newTaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), passphrase).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminTaintKeyParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
		PublicKey:  kp.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type taintKeyHandler struct {
	*api.AdminTaintKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *taintKeyHandler) handle(t *testing.T, ctx context.Context, params interface{}) *jsonrpc.ErrorDetails {
	t.Helper()

	result, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	assert.Nil(t, result)
	return err
}

func newTaintKeyHandler(t *testing.T) *taintKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &taintKeyHandler{
		AdminTaintKey: api.NewAdminTaintKey(walletStore),
		ctrl:          ctrl,
		walletStore:   walletStore,
	}
}
