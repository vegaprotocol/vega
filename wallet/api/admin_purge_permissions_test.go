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

func TestAdminPurgePermissions(t *testing.T) {
	t.Run("Purging permissions with invalid params fails", testPurgingPermissionsWithInvalidParamsFails)
	t.Run("Purging permissions with valid params succeeds", testPurgingPermissionsWithValidParamsSucceeds)
	t.Run("Purging permissions from wallet that does not exists fails", testPurgingPermissionsFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminPurgePermissionsGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminPurgePermissionsGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Getting internal error during wallet saving fails", testAdminPurgePermissionsGettingInternalErrorDuringWalletSavingFails)
}

func testPurgingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminPurgePermissionsParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminPurgePermissionsParams{
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
			handler := newPurgePermissionsHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testPurgingPermissionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	hostname1 := vgrand.RandomStr(5)
	if err := expectedWallet.UpdatePermissions(hostname1, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: "read",
			RestrictedKeys: []string{
				firstKey.PublicKey(),
			},
		},
	}); err != nil {
		t.Fatalf("could not update permissions for test: %v", err)
	}
	hostname2 := vgrand.RandomStr(5)
	if err := expectedWallet.UpdatePermissions(hostname2, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: "read",
			RestrictedKeys: []string{
				firstKey.PublicKey(),
			},
		},
	}); err != nil {
		t.Fatalf("could not update permissions for test: %v", err)
	}

	// setup
	handler := newPurgePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminPurgePermissionsParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, wallet.DefaultPermissions(), expectedWallet.Permissions(hostname1))
	assert.Equal(t, wallet.DefaultPermissions(), expectedWallet.Permissions(hostname2))
}

func testPurgingPermissionsFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newPurgePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminPurgePermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminPurgePermissionsGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newPurgePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminPurgePermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testAdminPurgePermissionsGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newPurgePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, name, passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminPurgePermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testAdminPurgePermissionsGettingInternalErrorDuringWalletSavingFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newPurgePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminPurgePermissionsParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type purgePermissionsHandler struct {
	*api.AdminPurgePermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *purgePermissionsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Empty(t, rawResult)
	return err
}

func newPurgePermissionsHandler(t *testing.T) *purgePermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &purgePermissionsHandler{
		AdminPurgePermissions: api.NewAdminPurgePermissions(walletStore),
		ctrl:                  ctrl,
		walletStore:           walletStore,
	}
}
