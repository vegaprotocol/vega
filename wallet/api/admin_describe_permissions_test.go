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

func TestAdminDescribePermissions(t *testing.T) {
	t.Run("Documentation matches the code", testAdminDescribePermissionsSchemaCorrect)
	t.Run("Describing permissions with invalid params fails", testAdminDescribingPermissionsWithInvalidParamsFails)
	t.Run("Describing permissions with valid params succeeds", testAdminDescribingPermissionsWithValidParamsSucceeds)
	t.Run("Describing permissions from wallet that does not exists fails", testAdminDescribingPermissionsFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminDescribePermissionsGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminDescribePermissionsGettingInternalErrorDuringWalletRetrievalFails)
}

func testAdminDescribePermissionsSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.describe_permissions", api.AdminDescribePermissionsParams{}, api.AdminDescribePermissionsResult{})
}

func testAdminDescribingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminDescribePermissionsParams{
				Wallet:   "",
				Hostname: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty hostname key",
			params: api.AdminDescribePermissionsParams{
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
			handler := newDescribePermissionsHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminDescribingPermissionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	if err := expectedWallet.UpdatePermissions(hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access: "read",
			AllowedKeys: []string{
				firstKey.PublicKey(),
			},
		},
	}); err != nil {
		t.Fatalf("could not update permissions for test: %v", err)
	}

	// setup
	handler := newDescribePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribePermissionsParams{
		Wallet:   expectedWallet.Name(),
		Hostname: hostname,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, api.AdminDescribePermissionsResult{
		Permissions: wallet.Permissions{
			PublicKeys: wallet.PublicKeysPermission{
				Access: "read",
				AllowedKeys: []string{
					firstKey.PublicKey(),
				},
			},
		},
	}, result)
}

func testAdminDescribingPermissionsFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribePermissionsParams{
		Wallet:   name,
		Hostname: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminDescribePermissionsGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribePermissionsParams{
		Wallet:   name,
		Hostname: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testAdminDescribePermissionsGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribePermissionsHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribePermissionsParams{
		Wallet:   name,
		Hostname: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

type describePermissionsHandler struct {
	*api.AdminDescribePermissions
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *describePermissionsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminDescribePermissionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminDescribePermissionsResult)
		if !ok {
			t.Fatal("AdminDescribePermissions handler result is not a AdminDescribePermissionsResult")
		}
		return result, err
	}
	return api.AdminDescribePermissionsResult{}, err
}

func newDescribePermissionsHandler(t *testing.T) *describePermissionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &describePermissionsHandler{
		AdminDescribePermissions: api.NewAdminDescribePermissions(walletStore),
		ctrl:                     ctrl,
		walletStore:              walletStore,
	}
}
