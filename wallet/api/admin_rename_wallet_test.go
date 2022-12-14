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

func TestAdminRenameWallet(t *testing.T) {
	t.Run("Renaming a wallet with invalid params fails", testRenamingWalletWithInvalidParamsFails)
	t.Run("Renaming a wallet with valid params succeeds", testRenamingWalletWithValidParamsSucceeds)
	t.Run("Renaming a wallet that does not exists fails", testRenamingWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during existing wallet verification does not rename the wallet", testGettingInternalErrorDuringExistingWalletVerificationDoesNotRenameWallet)
	t.Run("Renaming a wallet that with name that is already taken fails", testRenamingWalletWithNameAlreadyTakenFails)
	t.Run("Getting internal error during non-existing wallet verification does not rename the wallet", testGettingInternalErrorDuringNonExistingWalletVerificationDoesNotRenameWallet)
	t.Run("Getting internal error during renaming does not rename the wallet", testGettingInternalErrorDuringRenamingDoesNotRenameWallet)
}

func testRenamingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminRenameWalletParams{
				Wallet:  "",
				NewName: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty new name",
			params: api.AdminRenameWalletParams{
				Wallet:  vgrand.RandomStr(5),
				NewName: "",
			},
			expectedError: api.ErrNewNameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newRenameWalletHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testRenamingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, newName).Times(1).Return(false, nil)
	handler.walletStore.EXPECT().RenameWallet(ctx, name, newName).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameWalletParams{
		Wallet:  name,
		NewName: newName,
	})

	// then
	require.Nil(t, errorDetails)
}

func testRenamingWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameWalletParams{
		Wallet:  name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringExistingWalletVerificationDoesNotRenameWallet(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameWalletParams{
		Wallet:  name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testRenamingWalletWithNameAlreadyTakenFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, newName).Times(1).Return(true, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameWalletParams{
		Wallet:  name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletAlreadyExists)
}

func testGettingInternalErrorDuringNonExistingWalletVerificationDoesNotRenameWallet(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, newName).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameWalletParams{
		Wallet:  name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringRenamingDoesNotRenameWallet(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().WalletExists(ctx, newName).Times(1).Return(false, nil)
	handler.walletStore.EXPECT().RenameWallet(ctx, name, newName).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameWalletParams{
		Wallet:  name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not rename the wallet: %w", assert.AnError))
}

type renameWalletHandler struct {
	*api.AdminRenameWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *renameWalletHandler) handle(t *testing.T, ctx context.Context, params interface{}) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	require.Nil(t, rawResult)
	return err
}

func newRenameWalletHandler(t *testing.T) *renameWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &renameWalletHandler{
		AdminRenameWallet: api.NewAdminRenameWallet(walletStore),
		ctrl:              ctrl,
		walletStore:       walletStore,
	}
}
