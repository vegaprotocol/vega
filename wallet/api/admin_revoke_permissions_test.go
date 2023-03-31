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

func TestAdminRevokePermissions(t *testing.T) {
	t.Run("Revoking permissions with invalid params fails", testRevokingPermissionsWithInvalidParamsFails)
	t.Run("Revoking permissions with valid params succeeds", testRevokingPermissionsWithValidParamsSucceeds)
	t.Run("Revoking permissions from wallet that does not exists fails", testRevokingPermissionsFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminRevokePermissionsGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminRevokePermissionsGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Getting internal error during wallet saving fails", testAdminRevokePermissionsGettingInternalErrorDuringWalletSavingFails)
}

func testRevokingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminRevokePermissionsParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
				Hostname:   vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminRevokePermissionsParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: "",
				Hostname:   vgrand.RandomStr(5),
			},
			expectedError: api.ErrPassphraseIsRequired,
		}, {
			name: "with empty hostname",
			params: api.AdminRevokePermissionsParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: vgrand.RandomStr(5),
				Hostname:   "",
			},
			expectedError: api.ErrHostnameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newRevokePermissionsHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testRevokingPermissionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	hostname1 := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	if err := expectedWallet.UpdatePermissions(hostname1, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: "read",
			AllowedKeys: []string{
				firstKey.PublicKey(),
			},
		},
	}); err != nil {
		t.Fatalf("could not update permissions for test: %v", err)
	}
	hostname2 := vgrand.RandomStr(5)
	permissions2 := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: "read",
			AllowedKeys: []string{
				firstKey.PublicKey(),
			},
		},
	}
	if err := expectedWallet.UpdatePermissions(hostname2, permissions2); err != nil {
		t.Fatalf("could not update permissions for test: %v", err)
	}

	// setup
	handler := newRevokePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRevokePermissionsParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
		Hostname:   hostname1,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, wallet.DefaultPermissions(), expectedWallet.Permissions(hostname1))
	assert.Equal(t, permissions2, expectedWallet.Permissions(hostname2))
}

func testRevokingPermissionsFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newRevokePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRevokePermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
		Hostname:   vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminRevokePermissionsGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newRevokePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRevokePermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
		Hostname:   vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testAdminRevokePermissionsGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newRevokePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, name, passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRevokePermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
		Hostname:   vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testAdminRevokePermissionsGettingInternalErrorDuringWalletSavingFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newRevokePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRevokePermissionsParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
		Hostname:   vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type revokePermissionsHandler struct {
	*api.AdminRevokePermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *revokePermissionsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Empty(t, rawResult)
	return err
}

func newRevokePermissionsHandler(t *testing.T) *revokePermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &revokePermissionsHandler{
		AdminRevokePermissions: api.NewAdminRevokePermissions(walletStore),
		ctrl:                   ctrl,
		walletStore:            walletStore,
	}
}
