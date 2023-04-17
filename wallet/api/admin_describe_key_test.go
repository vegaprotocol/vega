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

func TestAdminDescribeKey(t *testing.T) {
	t.Run("Describing a key with invalid params fails", testAdminDescribingKeyWithInvalidParamsFails)
	t.Run("Describing a key with valid params succeeds", testAdminDescribingKeyWithValidParamsSucceeds)
	t.Run("Describing a key from wallet that does not exists fails", testAdminDescribeKeyDescribingKeyFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testAdminDescribeKeyGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testAdminDescribeKeyGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Describing a key that does not exists fails", testAdminDescribingKeyThatDoesNotExistsFails)
}

func testAdminDescribingKeyWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminDescribeKeyParams{
				Wallet:    "",
				PublicKey: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty public key",
			params: api.AdminDescribeKeyParams{
				Wallet:    vgrand.RandomStr(5),
				PublicKey: "",
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newDescribeKeyHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminDescribingKeyWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, firstKey := walletWithKey(t)

	// setup
	handler := newDescribeKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: firstKey.PublicKey(),
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, api.AdminDescribeKeyResult{
		PublicKey: firstKey.PublicKey(),
		Name:      firstKey.Name(),
		Algorithm: wallet.Algorithm{
			Name:    firstKey.AlgorithmName(),
			Version: firstKey.AlgorithmVersion(),
		},
		Metadata:  firstKey.Metadata(),
		IsTainted: firstKey.IsTainted(),
	}, result)
}

func testAdminDescribeKeyDescribingKeyFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeKeyParams{
		Wallet:    name,
		PublicKey: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminDescribeKeyGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeKeyParams{
		Wallet:    name,
		PublicKey: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testAdminDescribeKeyGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeKeyParams{
		Wallet:    name,
		PublicKey: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testAdminDescribingKeyThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet, _ := walletWithKey(t)

	// setup
	handler := newDescribeKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name()).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeKeyParams{
		Wallet:    expectedWallet.Name(),
		PublicKey: vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrPublicKeyDoesNotExist)
}

type describeKeyHandler struct {
	*api.AdminDescribeKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *describeKeyHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminDescribeKeyResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminDescribeKeyResult)
		if !ok {
			t.Fatal("AdminDescribeKey handler result is not a AdminDescribeKeyResult")
		}
		return result, err
	}
	return api.AdminDescribeKeyResult{}, err
}

func newDescribeKeyHandler(t *testing.T) *describeKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &describeKeyHandler{
		AdminDescribeKey: api.NewAdminDescribeKey(walletStore),
		ctrl:             ctrl,
		walletStore:      walletStore,
	}
}
