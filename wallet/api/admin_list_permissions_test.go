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

func TestAdminListPermissions(t *testing.T) {
	t.Run("Listing permissions with invalid params fails", testListingPermissionsWithInvalidParamsFails)
	t.Run("Listing permissions with valid params succeeds", testListingPermissionsWithValidParamsSucceeds)
	t.Run("Listing permissions from wallet that does not exists fails", testListingPermissionsFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminListPermissionsGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminListPermissionsGettingInternalErrorDuringWalletRetrievalFails)
}

func testListingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminListPermissionsParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminListPermissionsParams{
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
			handler := newListPermissionsHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testListingPermissionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	hostname := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	if err := expectedWallet.UpdatePermissions(hostname, wallet.Permissions{
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
	handler := newListPermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListPermissionsParams{
		Wallet:     expectedWallet.Name(),
		Passphrase: passphrase,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, api.AdminListPermissionsResult{
		Permissions: map[string]wallet.PermissionsSummary{
			hostname: {
				"public_keys": "read",
			},
		},
	}, result)
}

func testListingPermissionsFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newListPermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListPermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminListPermissionsGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newListPermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListPermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testAdminListPermissionsGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newListPermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, name, passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListPermissionsParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

type listPermissionsHandler struct {
	*api.AdminListPermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *listPermissionsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminListPermissionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminListPermissionsResult)
		if !ok {
			t.Fatal("AdminListPermissions handler result is not a AdminListPermissionsResult")
		}
		return result, err
	}
	return api.AdminListPermissionsResult{}, err
}

func newListPermissionsHandler(t *testing.T) *listPermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &listPermissionsHandler{
		AdminListPermissions: api.NewAdminListPermissions(walletStore),
		ctrl:                 ctrl,
		walletStore:          walletStore,
	}
}
