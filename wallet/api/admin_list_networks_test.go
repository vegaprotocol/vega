package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminListNetworks(t *testing.T) {
	t.Run("Listing networks succeeds", testListingNetworksSucceeds)
	t.Run("Getting internal error during listing fails", testGettingInternalErrorDuringListingNetworksFails)
}

func testListingNetworksSucceeds(t *testing.T) {
	// given
	expectedNetworks := []string{"fairground", "mainnnet", "local"}

	// setup
	handler := newListNetworksHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().ListNetworks().Times(1).Return(expectedNetworks, nil)

	// when
	result, errorDetails := handler.handle(t, context.Background(), nil)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, expectedNetworks, result.Networks)
}

func testGettingInternalErrorDuringListingNetworksFails(t *testing.T) {
	// setup
	handler := newListNetworksHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().ListNetworks().Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, context.Background(), nil)

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not list the networks: %w", assert.AnError))
}

type listNetworksHandler struct {
	*api.AdminListNetworks
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *listNetworksHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminListNetworksResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminListNetworksResult)
		if !ok {
			t.Fatal("AdminListWallets handler result is not a AdminListWalletsResult")
		}
		return result, err
	}
	return api.AdminListNetworksResult{}, err
}

func newListNetworksHandler(t *testing.T) *listNetworksHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &listNetworksHandler{
		AdminListNetworks: api.NewAdminListNetworks(networkStore),
		ctrl:              ctrl,
		networkStore:      networkStore,
	}
}
