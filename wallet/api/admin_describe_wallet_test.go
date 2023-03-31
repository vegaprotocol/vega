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

func TestAdminDescribeWallet(t *testing.T) {
	t.Run("Describing a wallet with invalid params fails", testDescribingWalletWithInvalidParamsFails)
	t.Run("Describing a wallet with valid params succeeds", testDescribingWalletWithValidParamsSucceeds)
	t.Run("Describing a wallet that does not exists fails", testDescribingWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during verification fails", testAdminDescribeWalletGettingInternalErrorDuringVerificationFails)
	t.Run("Getting internal error during retrieval fails", testAdminDescribeWalletGettingInternalErrorDuringRetrievalFails)
}

func testDescribingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminDescribeWalletParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminDescribeWalletParams{
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
			handler := newDescribeWalletHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testDescribingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, name, passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, api.AdminDescribeWalletResult{
		Name:                 expectedWallet.Name(),
		ID:                   expectedWallet.ID(),
		Type:                 expectedWallet.Type(),
		Version:              expectedWallet.KeyDerivationVersion(),
		KeyDerivationVersion: expectedWallet.KeyDerivationVersion(),
	}, result)
}

func testDescribingWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testAdminDescribeWalletGettingInternalErrorDuringVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testAdminDescribeWalletGettingInternalErrorDuringRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().UnlockWallet(ctx, name, passphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

type describeWalletHandler struct {
	*api.AdminDescribeWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *describeWalletHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminDescribeWalletResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminDescribeWalletResult)
		if !ok {
			t.Fatal("AdminDescribeWallet handler result is not a AdminDescribeWalletResult")
		}
		return result, err
	}
	return api.AdminDescribeWalletResult{}, err
}

func newDescribeWalletHandler(t *testing.T) *describeWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &describeWalletHandler{
		AdminDescribeWallet: api.NewAdminDescribeWallet(walletStore),
		ctrl:                ctrl,
		walletStore:         walletStore,
	}
}
