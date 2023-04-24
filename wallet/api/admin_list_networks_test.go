package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/network"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminListNetworks(t *testing.T) {
	t.Run("Documentation matches the code", testAdminListNetworksSchemaCorrect)
	t.Run("Listing networks succeeds", testListingNetworksSucceeds)
	t.Run("Getting internal error during listing fails", testGettingInternalErrorDuringListingNetworksFails)
}

func testAdminListNetworksSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.list_networks", nil, api.AdminListNetworksResult{})
}

func testListingNetworksSucceeds(t *testing.T) {
	// given
	fairground := &network.Network{
		Name: "fairground",
		Metadata: []network.Metadata{
			{
				Key:   "category",
				Value: "test",
			},
		},
	}
	mainnet := &network.Network{
		Name: "mainnet",
		Metadata: []network.Metadata{
			{
				Key:   "category",
				Value: "main",
			},
		},
	}
	local := &network.Network{
		Name: "local",
	}
	expectedNetworks := []api.AdminListNetworkResult{
		{
			Name:     fairground.Name,
			Metadata: fairground.Metadata,
		},
		{
			Name:     mainnet.Name,
			Metadata: mainnet.Metadata,
		},
		{
			Name:     local.Name,
			Metadata: local.Metadata,
		},
	}

	// setup
	handler := newListNetworksHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().ListNetworks().Times(1).Return([]string{"fairground", "mainnet", "local"}, nil)
	gomock.InOrder(
		handler.networkStore.EXPECT().GetNetwork("fairground").Times(1).Return(fairground, nil),
		handler.networkStore.EXPECT().GetNetwork("mainnet").Times(1).Return(mainnet, nil),
		handler.networkStore.EXPECT().GetNetwork("local").Times(1).Return(local, nil),
	)

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

func (h *listNetworksHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminListNetworksResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
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
