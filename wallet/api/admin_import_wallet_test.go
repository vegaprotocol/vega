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

func TestAdminImportWallet(t *testing.T) {
	t.Run("Importing a wallet with invalid params fails", testImportingWalletWithInvalidParamsFails)
	t.Run("Importing a wallet with valid params succeeds", testImportingWalletWithValidParamsSucceeds)
	t.Run("Importing a wallet that already exists fails", testImportingWalletThatAlreadyExistsFails)
	t.Run("Getting internal error during verification does not import the wallet", testGettingInternalErrorDuringVerificationDoesNotImportWallet)
	t.Run("Getting internal error during saving does not import the wallet", testGettingInternalErrorDuringSavingDoesNotImportWallet)
}

func testImportingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminImportWalletParams{
				Wallet:               "",
				RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
				KeyDerivationVersion: 2,
				Passphrase:           vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminImportWalletParams{
				Wallet:               vgrand.RandomStr(5),
				RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
				KeyDerivationVersion: 2,
				Passphrase:           "",
			},
			expectedError: api.ErrPassphraseIsRequired,
		}, {
			name: "with empty recovery phrase",
			params: api.AdminImportWalletParams{
				Wallet:               vgrand.RandomStr(5),
				RecoveryPhrase:       "",
				KeyDerivationVersion: 2,
				Passphrase:           vgrand.RandomStr(5),
			},
			expectedError: api.ErrRecoveryPhraseIsRequired,
		}, {
			name: "with unset key derivation version",
			params: api.AdminImportWalletParams{
				Wallet:               vgrand.RandomStr(5),
				RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
				KeyDerivationVersion: 0,
				Passphrase:           vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletKeyDerivationVersionIsRequired,
		}, {
			name: "with unset key derivation version",
			params: api.AdminImportWalletParams{
				Wallet:         vgrand.RandomStr(5),
				RecoveryPhrase: "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
				Version:        0,
				Passphrase:     vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletKeyDerivationVersionIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newImportWalletHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testImportingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	expectedPath := filepath.Join(vgrand.RandomStr(3), vgrand.RandomStr(3))
	var importedWallet wallet.Wallet

	// setup
	handler := newImportWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), passphrase).Times(1).DoAndReturn(func(_ context.Context, w wallet.Wallet, passphrase string) error {
		importedWallet = w
		return nil
	})
	handler.walletStore.EXPECT().GetWalletPath(name).Times(1).Return(expectedPath)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportWalletParams{
		Wallet:               name,
		Passphrase:           passphrase,
		RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
		KeyDerivationVersion: 2,
	})

	// then
	require.Nil(t, errorDetails)
	// Verify imported wallet.
	assert.Equal(t, name, importedWallet.Name())
	// Verify the first generated key.
	assert.Len(t, importedWallet.ListKeyPairs(), 1)
	keyPair := importedWallet.ListKeyPairs()[0]
	assert.Equal(t, []wallet.Metadata{{Key: "name", Value: "Key 1"}}, keyPair.Metadata())
	// Verify the result.
	assert.Equal(t, name, result.Wallet.Name)
	assert.Equal(t, uint32(2), result.Wallet.Version)
	assert.Equal(t, expectedPath, result.Wallet.FilePath)
	assert.Equal(t, keyPair.PublicKey(), result.Key.PublicKey)
	assert.Equal(t, keyPair.AlgorithmName(), result.Key.Algorithm.Name)
	assert.Equal(t, keyPair.AlgorithmVersion(), result.Key.Algorithm.Version)
	assert.Equal(t, keyPair.Metadata(), result.Key.Meta)
}

func testImportingWalletThatAlreadyExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newImportWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportWalletParams{
		Wallet:               name,
		Passphrase:           passphrase,
		RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
		KeyDerivationVersion: 2,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletAlreadyExists)
}

func testGettingInternalErrorDuringVerificationDoesNotImportWallet(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newImportWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportWalletParams{
		Wallet:               name,
		Passphrase:           passphrase,
		RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
		KeyDerivationVersion: 2,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringSavingDoesNotImportWallet(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newImportWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), passphrase).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminImportWalletParams{
		Wallet:               name,
		Passphrase:           passphrase,
		RecoveryPhrase:       "swing ceiling chaos green put insane ripple desk match tip melt usual shrug turkey renew icon parade veteran lens govern path rough page render",
		KeyDerivationVersion: 2,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet: %w", assert.AnError))
}

type importWalletHandler struct {
	*api.AdminImportWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *importWalletHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminImportWalletResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminImportWalletResult)
		if !ok {
			t.Fatal("AdminImportWallet handler result is not a AdminImportWalletResult")
		}
		return result, err
	}
	return api.AdminImportWalletResult{}, err
}

func newImportWalletHandler(t *testing.T) *importWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &importWalletHandler{
		AdminImportWallet: api.NewAdminImportWallet(walletStore),
		ctrl:              ctrl,
		walletStore:       walletStore,
	}
}
