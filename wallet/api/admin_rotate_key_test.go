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

func TestAdminRotateKey(t *testing.T) {
	t.Run("Rotating a key with invalid params fails", testRotatingKeyWithInvalidParamsFails)
	t.Run("Rotating a key with valid params succeeds", testRotatingKeyWithValidParamsSucceeds)
	t.Run("Rotating a key from wallet that does not exists fails", testRotatingKeyFromWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during wallet verification fails", testRotatingKeyGettingInternalErrorDuringWalletVerificationFails)
	t.Run("Getting internal error during wallet retrieval fails", testRotatingKeyGettingInternalErrorDuringWalletRetrievalFails)
	t.Run("Rotating key on an isolated wallet fails", testRotatingKeyWithIsolatedWalletFails)
	t.Run("Rotating a key from a public key that does not exists fails", testRotatingKeyFromPublicKeyThatDoesNotExistsFails)
	t.Run("Rotating a key to a public key that does not exists fails", testRotatingKeyToPublicKeyThatDoesNotExistsFails)
	t.Run("Rotating a key to a tainted public key that does not exists fails", testRotatingKeyToTaintedPublicKeyDoesNotExistsFails)
}

func testRotatingKeyWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminRotateKeyParams{
				Wallet:                "",
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            "",
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrPassphraseIsRequired,
		}, {
			name: "with empty chain ID",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               "",
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrChainIDIsRequired,
		}, {
			name: "with empty current public key",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrCurrentPublicKeyIsRequired,
		}, {
			name: "with empty next public key",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrNextPublicKeyIsRequired,
		}, {
			name: "with unset submission block height",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 0,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrSubmissionBlockHeightIsRequired,
		}, {
			name: "with unset enactment block height",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  0,
			},
			expectedError: api.ErrEnactmentBlockHeightIsRequired,
		}, {
			name: "with same next and current public key",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  15,
			},
			expectedError: api.ErrNextAndCurrentPublicKeysCannotBeTheSame,
		}, {
			name: "with equal block height for enactment and submission",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  10,
			},
			expectedError: api.ErrEnactmentBlockHeightMustBeGreaterThanSubmissionOne,
		}, {
			name: "with enactment block height lower than submission one",
			params: api.AdminRotateKeyParams{
				Wallet:                vgrand.RandomStr(5),
				Passphrase:            vgrand.RandomStr(5),
				FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
				ChainID:               vgrand.RandomStr(5),
				SubmissionBlockHeight: 10,
				EnactmentBlockHeight:  5,
			},
			expectedError: api.ErrEnactmentBlockHeightMustBeGreaterThanSubmissionOne,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newRotateKeyHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testRotatingKeyWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	secondKey := generateKey(t, expectedWallet)

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                expectedWallet.Name(),
		Passphrase:            passphrase,
		FromPublicKey:         firstKey.PublicKey(),
		ToPublicKey:           secondKey.PublicKey(),
		ChainID:               "test",
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.Nil(t, errorDetails)
	assert.NotEmpty(t, result)
}

func testRotatingKeyFromWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                name,
		Passphrase:            passphrase,
		FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
		ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testRotatingKeyGettingInternalErrorDuringWalletVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                name,
		Passphrase:            passphrase,
		FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
		ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testRotatingKeyGettingInternalErrorDuringWalletRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                name,
		Passphrase:            passphrase,
		FromPublicKey:         "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
		ToPublicKey:           "988eae323a07f12363c17025c23ee58ea32ac3912398e16bb0b56969f57adc52",
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testRotatingKeyWithIsolatedWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	w, firstKey := walletWithKey(t)
	secondKey := generateKey(t, w)
	isolatedWallet, err := w.IsolateWithKey(firstKey.PublicKey())
	if err != nil {
		t.Fatalf("could not isolate key for test: %v", err)
	}

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, isolatedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, isolatedWallet.Name(), passphrase).Times(1).Return(isolatedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                isolatedWallet.Name(),
		Passphrase:            passphrase,
		FromPublicKey:         firstKey.PublicKey(),
		ToPublicKey:           secondKey.PublicKey(),
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrCannotRotateKeysOnIsolatedWallet)
}

func testRotatingKeyFromPublicKeyThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)
	secondKey := generateKey(t, expectedWallet)

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                expectedWallet.Name(),
		Passphrase:            passphrase,
		FromPublicKey:         vgrand.RandomStr(5),
		ToPublicKey:           secondKey.PublicKey(),
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrCurrentPublicKeyDoesNotExist)
}

func testRotatingKeyToPublicKeyThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                expectedWallet.Name(),
		Passphrase:            passphrase,
		FromPublicKey:         firstKey.PublicKey(),
		ToPublicKey:           vgrand.RandomStr(5),
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrNextPublicKeyDoesNotExist)
}

func testRotatingKeyToTaintedPublicKeyDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	expectedWallet, firstKey := walletWithKey(t)
	secondKey := generateKey(t, expectedWallet)
	if err := expectedWallet.TaintKey(secondKey.PublicKey()); err != nil {
		t.Fatalf("could not taint the second key for test: %v", err)
	}

	// setup
	handler := newRotateKeyHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, expectedWallet.Name()).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, expectedWallet.Name(), passphrase).Times(1).Return(expectedWallet, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRotateKeyParams{
		Wallet:                expectedWallet.Name(),
		Passphrase:            passphrase,
		FromPublicKey:         firstKey.PublicKey(),
		ToPublicKey:           secondKey.PublicKey(),
		ChainID:               vgrand.RandomStr(5),
		SubmissionBlockHeight: 10,
		EnactmentBlockHeight:  15,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrNextPublicKeyIsTainted)
}

type rotateKeyHandler struct {
	*api.AdminRotateKey
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *rotateKeyHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminRotateKeyResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminRotateKeyResult)
		if !ok {
			t.Fatal("AdminRotateKey handler result is not a AdminRotateKeyResult")
		}
		return result, err
	}
	return api.AdminRotateKeyResult{}, err
}

func newRotateKeyHandler(t *testing.T) *rotateKeyHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &rotateKeyHandler{
		AdminRotateKey: api.NewAdminRotateKey(walletStore),
		ctrl:           ctrl,
		walletStore:    walletStore,
	}
}
