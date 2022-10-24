package api_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminIsolateKey(t *testing.T) {
	t.Run("Isolating a key with invalid params fails", testIsolatingKeyWithInvalidParamsFails)
	t.Run("Isolating a key with valid params succeeds", testIsolatingKeyWithValidParamsSucceeds)
	t.Run("Isolating a key from wallet that does not exists fails", testIsolatingKeyFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testIsolatingKeyGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testIsolatingKeyGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Isolating a key that does not exists fails", testIsolatingKeyThatDoesNotExistsFails)
	t.Run("Getting internal error during isolated wallet saving fails", testIsolatingKeyGettingInternalErrorDuringIsolatedWalletSavingFails)
}

func testIsolatingKeyWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminIsolateKeyParams{
				Wallet:                   "",
				Passphrase:               vgrand.RandomStr(5),
				PublicKey:                "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				IsolatedWalletPassphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminIsolateKeyParams{
				Wallet:                   vgrand.RandomStr(5),
				Passphrase:               "",
				PublicKey:                "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				IsolatedWalletPassphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrPassphraseIsRequired,
		}, {
			name: "with empty isolated passphrase",
			params: api.AdminIsolateKeyParams{
				Wallet:                   vgrand.RandomStr(5),
				Passphrase:               vgrand.RandomStr(5),
				IsolatedWalletPassphrase: "",
				PublicKey:                "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
			expectedError: api.ErrIsolatedWalletPassphraseIsRequired,
		}, {
			name: "with empty public key",
			params: api.AdminIsolateKeyParams{
				Wallet:                   vgrand.RandomStr(5),
				Passphrase:               vgrand.RandomStr(5),
				PublicKey:                "",
				IsolatedWalletPassphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrPublicKeyIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newIsolateKeyHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testIsolatingKeyWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	isolatedPassphrase := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	expectedPath := filepath.Join(vgrand.RandomStr(3), vgrand.RandomStr(3))

	// setup
	handler := newIsolateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), isolatedPassphrase).Times(1).Return(nil)
	handler.walletStore.EXPECT().GetWalletPath(gomock.Any()).Times(1).Return(expectedPath)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminIsolateKeyParams{
		Wallet:                   expectedWallet.Name(),
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: isolatedPassphrase,
		PublicKey:                firstKey.PublicKey(),
	})

	// then
	require.Nil(t, errorDetails)
	assert.True(t, strings.HasPrefix(result.Wallet, expectedWallet.Name()))
	assert.Equal(t, expectedPath, result.FilePath)
}

func testIsolatingKeyFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	isolatedPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newIsolateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminIsolateKeyParams{
		Wallet:                   name,
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: isolatedPassphrase,
		PublicKey:                vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testIsolatingKeyGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	isolatedPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newIsolateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminIsolateKeyParams{
		Wallet:                   name,
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: isolatedPassphrase,
		PublicKey:                vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testIsolatingKeyGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	isolatedPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newIsolateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminIsolateKeyParams{
		Wallet:                   name,
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: isolatedPassphrase,
		PublicKey:                vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testIsolatingKeyGettingInternalErrorDuringIsolatedWalletSavingFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	isolatedPassphrase := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)

	// setup
	handler := newIsolateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)
	handler.walletStore.EXPECT().SaveWallet(ctx, gomock.Any(), isolatedPassphrase).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminIsolateKeyParams{
		Wallet:                   expectedWallet.Name(),
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: isolatedPassphrase,
		PublicKey:                firstKey.PublicKey(),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the wallet with isolated key: %w", assert.AnError))
}

func testIsolatingKeyThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	isolatedPassphrase := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)

	// setup
	handler := newIsolateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminIsolateKeyParams{
		Wallet:                   expectedWallet.Name(),
		Passphrase:               passphrase,
		IsolatedWalletPassphrase: isolatedPassphrase,
		PublicKey:                vgrand.RandomStr(5),
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrPublicKeyDoesNotExist)
}

type isolateKeyHandler struct {
	*api.AdminIsolateKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *isolateKeyHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminIsolateKeyResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminIsolateKeyResult)
		if !ok {
			t.Fatal("AdminIsolateKey handler result is not a AdminIsolateKeyResult")
		}
		return result, err
	}
	return api.AdminIsolateKeyResult{}, err
}

func newIsolateKeyHandler(t *testing.T) *isolateKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &isolateKeyHandler{
		AdminIsolateKey: api.NewAdminIsolateKey(walletStore),
		ctrl:            ctrl,
		walletStore:     walletStore,
	}
}
