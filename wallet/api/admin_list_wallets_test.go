package api_test

import (
	"context"
	"fmt"
	"sort"
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

func TestAdminListWallet(t *testing.T) {
	t.Run("Listing wallets succeeds", testListingWalletsSucceeds)
	t.Run("Getting internal error during listing fails", testGettingInternalErrorDuringListingFails)
}

func testListingWalletsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedWallet1, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatal(fmt.Errorf("couldn't create wallet for test: %w", err))
	}
	expectedWallet2, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatal(fmt.Errorf("couldn't create wallet for test: %w", err))
	}

	// setup
	handler := newListWalletHandlers(t)
	// -- expected calls
	expectedWallets := []string{expectedWallet1.Name(), expectedWallet2.Name()}
	sort.Strings(expectedWallets)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(expectedWallets, nil)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.Nil(t, errorDetails)
	// Verify generated wallet.
	assert.Equal(t, expectedWallets, result.Wallets)
}

func testGettingInternalErrorDuringListingFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newListWalletHandlers(t)
	// -- expected calls
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(1).Return(nil, assert.AnError)
	// -- unexpected calls
	handler.walletStore.EXPECT().SaveWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().WalletExists(gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().GetWallet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	handler.walletStore.EXPECT().DeleteWallet(gomock.Any(), gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.NotNil(t, errorDetails)
	// Verify generated wallet.
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("couldn't list wallets: %w", assert.AnError))
}

type listWalletsHandler struct {
	*api.ListWallets
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
	pipeline    *mocks.MockPipeline
}

func (h *listWalletsHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.ListWalletsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.ListWalletsResult)
		if !ok {
			t.Fatal("ListWallets handler result is not a ListWalletsResult")
		}
		return result, err
	}
	return api.ListWalletsResult{}, err
}

func newListWalletHandlers(t *testing.T) *listWalletsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &listWalletsHandler{
		ListWallets: api.NewListWallets(walletStore),
		ctrl:        ctrl,
		walletStore: walletStore,
	}
}
