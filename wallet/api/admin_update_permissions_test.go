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

func TestAdminUpdatePermissions(t *testing.T) {
	t.Run("Updating permissions with invalid params fails", testUpdatingPermissionsWithInvalidParamsFails)
	t.Run("Updating permissions with valid params succeeds", testUpdatingPermissionsWithValidParamsSucceeds)
	t.Run("Updating permissions from wallet that does not exists fails", testUpdatingPermissionsFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminUpdatePermissionsGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminUpdatePermissionsGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Getting internal error during wallet saving fails", testAdminUpdatePermissionsGettingInternalErrorDuringWalletSavingFails)
}

func testUpdatingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminUpdatePermissionsParams{
				Wallet:   "",
				Hostname: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty hostname",
			params: api.AdminUpdatePermissionsParams{
				Wallet:   vgrand.RandomStr(5),
				Hostname: "",
			},
			expectedError: api.ErrHostnameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newUpdatePermissionsHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

func testUpdatingPermissionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	permissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: "read",
			AllowedKeys: []string{
				firstKey.PublicKey(),
			},
		},
	}

	// setup
	handler := newUpdatePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminUpdatePermissionsParams{
		Wallet:      expectedWallet.Name(),
		Hostname:    hostname,
		Permissions: permissions,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, permissions, result.Permissions)
	assert.Equal(t, permissions, expectedWallet.Permissions(hostname))
}

func testUpdatingPermissionsFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdatePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminUpdatePermissionsParams{
		Wallet:   name,
		Hostname: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	require.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminUpdatePermissionsGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdatePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminUpdatePermissionsParams{
		Wallet:   name,
		Hostname: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	require.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testAdminUpdatePermissionsGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdatePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminUpdatePermissionsParams{
		Wallet:   name,
		Hostname: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	require.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testAdminUpdatePermissionsGettingInternalErrorDuringWalletSavingFails(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newUpdatePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminUpdatePermissionsParams{
		Wallet:      expectedWallet.Name(),
		Hostname:    vgrand.RandomStr(5),
		Permissions: wallet.Permissions{},
	})

	// then
	require.NotNil(t, errorDetails)
	require.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type updatePermissionsHandler struct {
	*api.AdminUpdatePermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *updatePermissionsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminUpdatePermissionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminUpdatePermissionsResult)
		if !ok {
			t.Fatal("AdminUpdatePermissions handler result is not a AdminUpdatePermissionsResult")
		}
		return result, err
	}
	return api.AdminUpdatePermissionsResult{}, err
}

func newUpdatePermissionsHandler(t *testing.T) *updatePermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &updatePermissionsHandler{
		AdminUpdatePermissions: api.NewAdminUpdatePermissions(walletStore),
		ctrl:                   ctrl,
		walletStore:            walletStore,
	}
}
