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
	t.Run("Generating a key with invalid params fails", testGeneratingKeyWithInvalidParamsFails)
	t.Run("Generating a key with valid params succeeds", testGeneratingKeyWithValidParamsSucceeds)
	t.Run("Generating a key on unknown wallet fails", testGeneratingKeyOnUnknownWalletFails)
	t.Run("Getting internal error during wallet verification doesn't import the wallet", testGettingInternalErrorDuringWalletVerificationDoesNotGenerateKey)
	t.Run("Getting internal error during wallet retrieval doesn't import the wallet", testGettingInternalErrorDuringWalletRetrievalDoesNotGenerateKey)
	t.Run("Getting internal error during wallet saving doesn't import the wallet", testGettingInternalErrorDuringWalletSavingDoesNotGenerateKey)
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
			params: api.GenerateKeyParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.GenerateKeyParams{
				Wallet:     vgrand.RandomStr(5),
				Passphrase: "",
			},
			expectedError: api.ErrPassphraseIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newGenerateKeyHandler(tt)
			// -- unexpected calls
			handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
			handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

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
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, expectedWallet, passphrase).Times(1).Return(nil)
	// -- unexpected calls
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GenerateKeyParams{
		Wallet:     name,
		Passphrase: passphrase,
		Metadata:   []wallet.Meta{{Key: "mode", Value: "test"}},
	})

	// then
	require.Nil(t, errorDetails)
	// Verify generated wallet.
	assert.Equal(t, name, expectedWallet.Name())
	// Verify the first generated key.
	assert.Len(t, expectedWallet.ListKeyPairs(), 1)
	keyPair := expectedWallet.ListKeyPairs()[0]
	assert.Equal(t, []wallet.Meta{{Key: "mode", Value: "test"}, {Key: "name", Value: fmt.Sprintf("%s key 1", name)}}, keyPair.Meta())
	// Verify the result.
	assert.Equal(t, keyPair.PublicKey(), result.PublicKey)
	assert.Equal(t, keyPair.AlgorithmName(), result.Algorithm.Name)
	assert.Equal(t, keyPair.AlgorithmVersion(), result.Algorithm.Version)
	assert.Equal(t, keyPair.Meta(), result.Metadata)
}

func testGeneratingKeyOnUnknownWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GenerateKeyParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	// Verify generated wallet.
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotGenerateKey(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GenerateKeyParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("couldn't verify wallet existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotGenerateKey(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(nil, assert.AnError)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GenerateKeyParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	// Verify generated wallet.
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("couldn't retrieve wallet: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletSavingDoesNotGenerateKey(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newGenerateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), passphrase).Times(1).Return(assert.AnError)
	// -- unexpected calls
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GenerateKeyParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("couldn't save wallet: %w", assert.AnError))
}

type generateKeyHandler struct {
	*api.GenerateKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	pipeline    *mocks.MockPipeline
}

func (h *generateKeyHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.GenerateKeyResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.GenerateKeyResult)
		if !ok {
			t.Fatal("GenerateKey handler result is not a GenerateKeyResult")
		}
		return result, err
	}
	return api.GenerateKeyResult{}, err
}

func newGenerateKeyHandler(t *testing.T) *generateKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &generateKeyHandler{
		GenerateKey: api.NewGenerateKey(walletStore),
		ctrl:        ctrl,
		walletStore: walletStore,
	}
}
