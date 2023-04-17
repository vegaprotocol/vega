package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUnlockWallet(t *testing.T) {
	t.Run("Unlocking a key with invalid params fails", testUnlockingWalletWithInvalidParamsFails)
	t.Run("Unlocking a key with valid params succeeds", testUnlockingWalletWithValidParamsSucceeds)
	t.Run("Unlocking an unknown wallet fails", testUnlockingUnknownWalletFails)
	t.Run("Getting internal error during wallet verification doesn't unlock the wallet", testGettingInternalErrorDuringWalletVerificationDoesNotUnlockWallet)
	t.Run("Unlocking the wallet with wrong passphrase fails", testUnlockingWalletWithWrongPassphraseFails)
	t.Run("Getting internal error during wallet unlocking doesn't unlock the wallet", testGettingInternalErrorDuringWalletUnlockingDoesNotUnlockWallet)
}

func testUnlockingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminUnlockWalletParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminUnlockWalletParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: "",
			},
			expectedError: api.ErrPassphraseIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newUnlockWalletHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testUnlockingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newUnlockWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet, passphrase).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUnlockWalletParams{
		Wallet:     expectedWallet,
		Passphrase: passphrase,
	})

	// then
	require.Nil(t, errorDetails)
}

func testUnlockingUnknownWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newUnlockWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUnlockWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotUnlockWallet(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet := vgrand.RandomStr(5)
	passphrase := vgrand.RandomStr(5)

	// setup
	handler := newUnlockWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUnlockWalletParams{
		Wallet:     expectedWallet,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testUnlockingWalletWithWrongPassphraseFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet := vgrand.RandomStr(5)

	// setup
	handler := newUnlockWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet, passphrase).Times(1).Return(wallet.ErrWrongPassphrase)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUnlockWalletParams{
		Wallet:     expectedWallet,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, wallet.ErrWrongPassphrase)
}

func testGettingInternalErrorDuringWalletUnlockingDoesNotUnlockWallet(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet := vgrand.RandomStr(5)

	// setup
	handler := newUnlockWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet, passphrase).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUnlockWalletParams{
		Wallet:     expectedWallet,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not unlock the wallet: %w", assert.AnError))
}

type unlockWalletHandler struct {
	*api.AdminUnlockWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *unlockWalletHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()
	rawResult, err := h.Handle(ctx, params)
	require.Nil(t, rawResult)
	return err
}

func newUnlockWalletHandler(t *testing.T) *unlockWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &unlockWalletHandler{
		AdminUnlockWallet: api.NewAdminUnlockWallet(walletStore),
		ctrl:              ctrl,
		walletStore:       walletStore,
	}
}
