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

func TestAdminGenerateKey(t *testing.T) {
	t.Run("Documentation matches the code", testAdminGenerateKeySchemaCorrect)
	t.Run("Generating a key with invalid params fails", testGeneratingKeyWithInvalidParamsFails)
	t.Run("Generating a key with valid params succeeds", testGeneratingKeyWithValidParamsSucceeds)
	t.Run("Generating a key on unknown wallet fails", testGeneratingKeyOnUnknownWalletFails)
	t.Run("Getting internal error during wallet verification doesn't generate the key", testGettingInternalErrorDuringWalletVerificationDoesNotGenerateKey)
	t.Run("Getting internal error during wallet retrieval doesn't generate the key", testGettingInternalErrorDuringWalletRetrievalDoesNotGenerateKey)
	t.Run("Getting internal error during wallet saving doesn't generate the key", testGettingInternalErrorDuringWalletSavingDoesNotGenerateKey)
}

func testAdminGenerateKeySchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.generate_key", api.AdminGenerateKeyParams{}, api.AdminGenerateKeyResult{})
}

func testGeneratingKeyWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminGenerateKeyParams{
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
			handler := newGenerateKeyHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testGeneratingKeyWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, expectedWallet).Times(1).Return(nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateKeyParams{
		Wallet:   name,
		Metadata: []wallet.Metadata{{Key: "mode", Value: "test"}},
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, name, expectedWallet.Name())
	assert.Len(t, expectedWallet.ListKeyPairs(), 1)
	keyPair := expectedWallet.ListKeyPairs()[0]
	assert.Equal(t, []wallet.Metadata{{Key: "mode", Value: "test"}, {Key: "name", Value: "Key 1"}}, keyPair.Metadata())
	// Verify the result.
	assert.Equal(t, keyPair.PublicKey(), result.PublicKey)
	assert.Equal(t, keyPair.AlgorithmName(), result.Algorithm.Name)
	assert.Equal(t, keyPair.AlgorithmVersion(), result.Algorithm.Version)
	assert.Equal(t, keyPair.Metadata(), result.Metadata)
}

func testGeneratingKeyOnUnknownWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateKeyParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotGenerateKey(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateKeyParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotGenerateKey(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateKeyParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletSavingDoesNotGenerateKey(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().IsWalletAlreadyUnlocked(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().UpdateWallet(ctx, gomock.Any()).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateKeyParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type generateKeyHandler struct {
	*api.AdminGenerateKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *generateKeyHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminGenerateKeyResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminGenerateKeyResult)
		if !ok {
			t.Fatal("AdminGenerateKey handler result is not a AdminGenerateKeyResult")
		}
		return result, err
	}
	return api.AdminGenerateKeyResult{}, err
}

func newGenerateKeyHandler(t *testing.T) *generateKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &generateKeyHandler{
		AdminGenerateKey: api.NewAdminGenerateKey(walletStore),
		ctrl:             ctrl,
		walletStore:      walletStore,
	}
}
