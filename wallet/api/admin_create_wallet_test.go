package api_test

import (
	"context"
	"fmt"
	"path/filepath"
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

func TestAdminCreateWallet(t *testing.T) {
	t.Run("Creating a wallet with invalid params fails", testCreatingWalletWithInvalidParamsFails)
	t.Run("Creating a wallet with valid params succeeds", testCreatingWalletWithValidParamsSucceeds)
	t.Run("Creating a wallet that already exists fails", testCreatingWalletThatAlreadyExistsFails)
	t.Run("Getting internal error during verification does not create the wallet", testGettingInternalErrorDuringVerificationDoesNotCreateWallet)
	t.Run("Getting internal error during saving does not create the wallet", testGettingInternalErrorDuringSavingDoesNotCreateWallet)
}

func testCreatingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminCreateWalletParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminCreateWalletParams{
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
			handler := newCreateWalletHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testCreatingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	expectedPath := filepath.Join(vgrand.RandomStr(3), vgrand.RandomStr(3))
	var createdWallet wallet.Wallet

	// setup
	handler := newCreateWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)
	handler.walletStore.EXPECT().CreateWallet(ctx, gomock.Any(), passphrase).Times(1).DoAndReturn(func(_ context.Context, w wallet.Wallet, passphrase string) error {
		createdWallet = w
		return nil
	})
	handler.walletStore.EXPECT().GetWalletPath(name).Times(1).Return(expectedPath)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminCreateWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.Nil(t, errorDetails)
	// Verify generated wallet.
	assert.Equal(t, name, createdWallet.Name())
	// Verify the first generated key.
	assert.Len(t, createdWallet.ListKeyPairs(), 1)
	keyPair := createdWallet.ListKeyPairs()[0]
	assert.Equal(t, []wallet.Metadata{{Key: "name", Value: "Key 1"}}, keyPair.Metadata())
	// Verify the result.
	assert.Equal(t, name, result.Wallet.Name)
	assert.NotEmpty(t, result.Wallet.RecoveryPhrase)
	assert.Equal(t, uint32(2), result.Wallet.KeyDerivationVersion)
	assert.Equal(t, expectedPath, result.Wallet.FilePath)
	assert.Equal(t, keyPair.PublicKey(), result.Key.PublicKey)
	assert.Equal(t, keyPair.AlgorithmName(), result.Key.Algorithm.Name)
	assert.Equal(t, keyPair.AlgorithmVersion(), result.Key.Algorithm.Version)
	assert.Equal(t, keyPair.Metadata(), result.Key.Meta)
}

func testCreatingWalletThatAlreadyExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newCreateWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminCreateWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletAlreadyExists)
}

func testGettingInternalErrorDuringVerificationDoesNotCreateWallet(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newCreateWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminCreateWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

func testGettingInternalErrorDuringSavingDoesNotCreateWallet(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newCreateWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)
	handler.walletStore.EXPECT().CreateWallet(ctx, gomock.Any(), passphrase).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminCreateWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type createWalletHandler struct {
	*api.AdminCreateWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *createWalletHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminCreateWalletResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminCreateWalletResult)
		if !ok {
			t.Fatal("AdminCreateWallet handler result is not a AdminCreateWalletResult")
		}
		return result, err
	}
	return api.AdminCreateWalletResult{}, err
}

func newCreateWalletHandler(t *testing.T) *createWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &createWalletHandler{
		AdminCreateWallet: api.NewAdminCreateWallet(walletStore),
		ctrl:              ctrl,
		walletStore:       walletStore,
	}
}
