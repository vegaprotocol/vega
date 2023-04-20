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

func TestAdminAdminListKeys(t *testing.T) {
	t.Run("Documentation matches the code", testAdminListKeysSchemaCorrect)
	t.Run("Listing the keys with invalid params fails", testAdminListKeysWithInvalidParamsFails)
	t.Run("Listing the keys with valid params succeeds", testAdminListKeysWithValidParamsSucceeds)
	t.Run("Listing the keys from wallet that does not exists fails", testAdminListKeysFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminListKeysGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminListKeysGettingInternalErrorDuringWalletRetrievalFails)
}

func testAdminListKeysSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.list_keys", api.AdminListKeysParams{}, api.AdminListKeysResult{})
}

func testAdminListKeysWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminListKeysParams{
				Wallet: "",
			},
			expectedError: api.ErrWalletIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newAdminListKeysHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminListKeysWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)

	// setup
	handler := newAdminListKeysHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListKeysParams{
		Wallet: name,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, api.AdminListKeysResult{
		PublicKeys: []api.AdminNamedPublicKey{{
			Name:      firstKey.Name(),
			PublicKey: firstKey.PublicKey(),
		}},
	}, result)
}

func testAdminListKeysFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newAdminListKeysHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListKeysParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminListKeysGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newAdminListKeysHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListKeysParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testAdminListKeysGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newAdminListKeysHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminListKeysParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

type adminListKeysHandler struct {
	*api.AdminListKeys
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *adminListKeysHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminListKeysResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminListKeysResult)
		if !ok {
			t.Fatal("AdminListKeys handler result is not a AdminListKeysResult")
		}
		return result, err
	}
	return api.AdminListKeysResult{}, err
}

func newAdminListKeysHandler(t *testing.T) *adminListKeysHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &adminListKeysHandler{
		AdminListKeys: api.NewAdminListKeys(walletStore),
		ctrl:          ctrl,
		walletStore:   walletStore,
	}
}
