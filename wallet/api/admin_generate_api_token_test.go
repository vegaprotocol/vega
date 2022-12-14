package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminGenerateAPIToken(t *testing.T) {
	t.Run("Generating an API token with invalid params fails", testGeneratingAPITokenWithInvalidParamsFails)
	t.Run("Generating an API token with valid params succeeds", testGeneratingAPITokenWithValidParamsSucceeds)
	t.Run("Generating an API token with valid params past expiry fails", testGeneratingAPITokenWithValidParamsPastExpiryFails)
	t.Run("Generating an API token with valid params valid expiry succeeds", testGeneratingAPITokenWithValidParamsValidExpirySucceeds)
	t.Run("Generating an API token on unknown wallet fails", testGeneratingAPITokenOnUnknownWalletFails)
	t.Run("Getting internal error during wallet verification doesn't generate the token", testGettingInternalErrorDuringWalletVerificationDoesNotGenerateAPIToken)
	t.Run("Getting internal error during wallet retrieval doesn't generate the token", testGettingInternalErrorDuringWalletRetrievalDoesNotGenerateAPIToken)
	t.Run("Getting internal error during wallet saving doesn't generate the token", testGettingInternalErrorDuringTokenSavingDoesNotGenerateAPIToken)
}

func testGeneratingAPITokenWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty wallet name",
			params: api.AdminGenerateAPITokenParams{
				Wallet: api.AdminGenerateAPITokenWalletParams{
					Name:       "",
					Passphrase: vgrand.RandomStr(10),
				},
			},
			expectedError: api.ErrWalletNameIsRequired,
		}, {
			name: "with empty wallet name",
			params: api.AdminGenerateAPITokenParams{
				Wallet: api.AdminGenerateAPITokenWalletParams{
					Name:       vgrand.RandomStr(5),
					Passphrase: "",
				},
			},
			expectedError: api.ErrWalletPassphraseIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newGenerateAPITokenHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testGeneratingAPITokenWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	description := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, walletPassphrase).Times(1).Return(expectedWallet, nil)
	handler.tokenStore.EXPECT().SaveToken(gomock.Any()).Times(1).Return(nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
		Description: description,
	})

	// then
	require.Nil(t, errorDetails)
	assert.NotEmpty(t, result.Token)
}

func testGeneratingAPITokenWithValidParamsValidExpirySucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	description := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	okExpiry := now.Add(1 * time.Hour)

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, walletPassphrase).Times(1).Return(expectedWallet, nil)
	handler.tokenStore.EXPECT().SaveToken(gomock.Any()).Times(1).Return(nil)
	handler.time.EXPECT().Now().Times(1).Return(now)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
		ExpiryTs:    ptr.From(okExpiry.Unix()),
		Description: description,
	})

	// then
	require.Nil(t, errorDetails)
	assert.NotEmpty(t, result.Token)
}

func testGeneratingAPITokenWithValidParamsPastExpiryFails(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	description := vgrand.RandomStr(5)
	_, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	pastNow := now.Add(-1 * time.Hour)

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.time.EXPECT().Now().Times(1).Return(now)

	// when
	_, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
		ExpiryTs:    ptr.From(pastNow.Unix()),
		Description: description,
	})

	// then
	require.EqualError(t, errorDetails, "the token expiration date cannot be set to a past date (Invalid params -32602)")
}

func testGeneratingAPITokenOnUnknownWalletFails(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringWalletVerificationDoesNotGenerateAPIToken(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringWalletRetrievalDoesNotGenerateAPIToken(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, walletPassphrase).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the wallet: %w", assert.AnError))
}

func testGettingInternalErrorDuringTokenSavingDoesNotGenerateAPIToken(t *testing.T) {
	// given
	ctx := context.Background()
	walletPassphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)
	expectedWallet, _, err := wallet.NewHDWallet(name)
	if err != nil {
		t.Fatal(err)
	}

	// setup
	handler := newGenerateAPITokenHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, walletPassphrase).Times(1).Return(expectedWallet, nil)
	handler.tokenStore.EXPECT().SaveToken(gomock.Any()).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminGenerateAPITokenParams{
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       name,
			Passphrase: walletPassphrase,
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the newly generated token: %w", assert.AnError))
}

type generateAPITokenHandler struct {
	*api.AdminGenerateAPIToken
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	tokenStore  *mocks.MockTokenStore
	time        *mocks.MockTimeProvider
}

func (h *generateAPITokenHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminGenerateAPITokenResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminGenerateAPITokenResult)
		if !ok {
			t.Fatal("AdminGenerateAPIToken handler result is not a AdminGenerateAPITokenResult")
		}
		return result, err
	}
	return api.AdminGenerateAPITokenResult{}, err
}

func newGenerateAPITokenHandler(t *testing.T) *generateAPITokenHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	tp := mocks.NewMockTimeProvider(ctrl)

	return &generateAPITokenHandler{
		AdminGenerateAPIToken: api.NewAdminGenerateAPIToken(walletStore, tokenStore, tp),
		ctrl:                  ctrl,
		walletStore:           walletStore,
		tokenStore:            tokenStore,
		time:                  tp,
	}
}
