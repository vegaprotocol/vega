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
		t.Fatalf("could not create wallet for test: %v", err)
	}
	expectedWallet2, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}

	// setup
	handler := newListWalletHandlers(t)
	// -- expected calls
	expectedWallets := []string{expectedWallet1.Name(), expectedWallet2.Name()}
	sort.Strings(expectedWallets)
	handler.walletStore.EXPECT().ListWallets(ctx).Times(1).Return(expectedWallets, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, expectedWallets, result.Wallets)
}

func testGettingInternalErrorDuringListingFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newListWalletHandlers(t)
	// -- expected calls
	handler.walletStore.EXPECT().ListWallets(gomock.Any()).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not list the wallets: %w", assert.AnError))
}

type listWalletsHandler struct {
	*api.AdminListWallets
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *listWalletsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminListWalletsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminListWalletsResult)
		if !ok {
			t.Fatal("AdminListWallets handler result is not a AdminListWalletsResult")
		}
		return result, err
	}
	return api.AdminListWalletsResult{}, err
}

func newListWalletHandlers(t *testing.T) *listWalletsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &listWalletsHandler{
		AdminListWallets: api.NewAdminListWallets(walletStore),
		ctrl:             ctrl,
		walletStore:      walletStore,
	}
}
