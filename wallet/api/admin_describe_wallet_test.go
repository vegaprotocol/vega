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
	t.Run("Describing a wallet that doesn't exists fails", testDescribingWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during verification fails", testGettingInternalErrorDuringVerificationFails)
	t.Run("Getting internal error during retrieval fails", testGettingInternalErrorDuringRetrievalFails)
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
			params: api.DescribeWalletParams{
				Wallet:     "",
				Passphrase: vgrand.RandomStr(5),
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty passphrase",
			params: api.DescribeWalletParams{
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
			handler := newDescribeWalletHandler(tt)
			// -- unexpected calls
			handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)
			handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

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
		t.Fatal(fmt.Errorf("couldn't create wallet for test: %w", err))
	}

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(expectedWallet, nil)
	// -- unexpected calls
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.DescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.Nil(t, errorDetails)
	// Verify generated wallet.
	assert.Equal(t, api.DescribeWalletResult{
		Name:    expectedWallet.Name(),
		ID:      expectedWallet.ID(),
		Type:    expectedWallet.Type(),
		Version: expectedWallet.Version(),
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
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.DescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	// Verify generated wallet.
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)
	// -- unexpected calls
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.DescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	// Verify generated wallet.
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("couldn't verify wallet existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	passphrase := vgrand.RandomStr(5)
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().GetWallet(ctx, name, passphrase).Times(1).Return(nil, assert.AnError)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.DescribeWalletParams{
		Wallet:     name,
		Passphrase: passphrase,
	})

	// then
	require.NotNil(t, errorDetails)
	// Verify generated wallet.
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("couldn't retrieve wallet: %w", assert.AnError))
}

type describeWalletHandler struct {
	*api.DescribeWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	pipeline    *mocks.MockPipeline
}

func (h *describeWalletHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.DescribeWalletResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.DescribeWalletResult)
		if !ok {
			t.Fatal("DescribeWallet handler result is not a DescribeWalletResult")
		}
		return result, err
	}
	return api.DescribeWalletResult{}, err
}

func newDescribeWalletHandler(t *testing.T) *describeWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &describeWalletHandler{
		DescribeWallet: api.NewDescribeWallet(walletStore),
		ctrl:           ctrl,
		walletStore:    walletStore,
	}
}
