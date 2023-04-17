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

func TestAdminUntaintKey(t *testing.T) {
	t.Run("Untainting a key with invalid params fails", testUntaintingKeyWithInvalidParamsFails)
	t.Run("Untainting a key with valid params succeeds", testUntaintingKeyWithValidParamsSucceeds)
	t.Run("Untainting a key on unknown wallet fails", testUntaintingKeyOnUnknownWalletFails)
	t.Run("Untainting a key on unknown key fails", testUntaintingKeyOnUnknownKeyFails)
	t.Run("Getting internal error during wallet verification doesn't remove the taint", testGettingInternalErrorDuringWalletVerificationDoesNotUntaintKey)
	t.Run("Getting internal error during wallet retrieval doesn't remove the taint", testGettingInternalErrorDuringWalletRetrievalDoesNotUntaintKey)
	t.Run("Getting internal error during wallet saving doesn't remove the taint", testGettingInternalErrorDuringWalletSavingDoesNotUntaintKey)
}

func testUntaintingKeyWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminUntaintKeyParams{
				Wallet:    "",
				PublicKey: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty public key",
			params: api.AdminUntaintKeyParams{
				PublicKey: "",
				Wallet:    vgrand.RandomStr(5),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newUntaintKeyHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// the
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testUntaintingKeyWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, kp := walletWithKey(t)
	if err := expectedWallet.TaintKey(kp.PublicKey()); err != nil {
		t.Fatalf("could not taint the key for test: %v", err)
	}

	// setup
	handler := newUntaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUntaintKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: kp.PublicKey(),
	})

	// then
	require.Nil(t, errorDetails)
	require.False(t, expectedWallet.ListKeyPairs()[0].IsTainted())
}

func testUntaintingKeyOnUnknownWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUntaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUntaintKeyParams{
		Wallet:    name,
		PublicKey: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testUntaintingKeyOnUnknownKeyFails(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, _ := walletWithKey(t)

	// setup
	handler := newUntaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUntaintKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrPublicKeyDoesNotExist)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotUntaintKey(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, kp := walletWithKey(t)

	// setup
	handler := newUntaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUntaintKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: kp.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotUntaintKey(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, kp := walletWithKey(t)

	// setup
	handler := newUntaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(nil, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUntaintKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: kp.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletSavingDoesNotUntaintKey(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, kp := walletWithKey(t)
	if err := expectedWallet.TaintKey(kp.PublicKey()); err != nil {
		t.Fatalf("could not taint the key for test: %v", err)
	}

	// setup
	handler := newUntaintKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, gomock.Any()).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUntaintKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: kp.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type untaintKeyHandler struct {
	*api.AdminUntaintKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *untaintKeyHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	result, err := h.Handle(ctx, params)
	assert.Nil(t, result)
	return err
}

func newUntaintKeyHandler(t *testing.T) *untaintKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &untaintKeyHandler{
		AdminUntaintKey: api.NewAdminUntaintKey(walletStore),
		ctrl:            ctrl,
		walletStore:     walletStore,
	}
}
