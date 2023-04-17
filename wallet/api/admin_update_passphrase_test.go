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

func TestAdminUpdatePassphrase(t *testing.T) {
	t.Run("Updating a passphrase with invalid params fails", testUpdatingPassphraseWithInvalidParamsFails)
	t.Run("Updating a passphrase with valid params succeeds", testUpdatingPassphraseWithValidParamsSucceeds)
	t.Run("Getting internal error during wallet verification fails", testUpdatingPassphraseGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Updating a passphrase from wallet that does not exists fails", testUpdatingPassphraseFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during isolated wallet saving fails", testUpdatingPassphraseGettingInternalErrorDuringPassphraseUpdateFails)
}

func testUpdatingPassphraseWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminUpdatePassphraseParams{
				Wallet:        "",
				NewPassphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty new passphrase",
			params: api.AdminUpdatePassphraseParams{
				Wallet:        vgrand.RandomStr(5),
				NewPassphrase: "",
			},
			expectedError: api.ErrNewPassphraseIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newUpdatePassphraseHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testUpdatingPassphraseWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	newPassphrase := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)

	// setup
	handler := newUpdatePassphraseHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UpdatePassphrase(ctx, expectedWallet.Name(), newPassphrase).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUpdatePassphraseParams{
		Wallet:        expectedWallet.Name(),
		NewPassphrase: newPassphrase,
	})

	// then
	require.Nil(t, errorDetails)
}

func testUpdatingPassphraseFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	newPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdatePassphraseHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUpdatePassphraseParams{
		Wallet:        name,
		NewPassphrase: newPassphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testUpdatingPassphraseGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	newPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdatePassphraseHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUpdatePassphraseParams{
		Wallet:        name,
		NewPassphrase: newPassphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testUpdatingPassphraseGettingInternalErrorDuringPassphraseUpdateFails(t *testing.T) {
	// given
	ctx := context.Background()
	newPassphrase := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)

	// setup
	handler := newUpdatePassphraseHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UpdatePassphrase(ctx, gomock.Any(), newPassphrase).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUpdatePassphraseParams{
		Wallet:        expectedWallet.Name(),
		NewPassphrase: newPassphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet with the new passphrase: %w", assert.AnError))
}

type updatePassphraseHandler struct {
	*api.AdminUpdatePassphrase
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *updatePassphraseHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Nil(t, rawResult)
	return err
}

func newUpdatePassphraseHandler(t *testing.T) *updatePassphraseHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &updatePassphraseHandler{
		AdminUpdatePassphrase: api.NewAdminUpdatePassphrase(walletStore),
		ctrl:                  ctrl,
		walletStore:           walletStore,
	}
}
