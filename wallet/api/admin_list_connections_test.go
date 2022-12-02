package api_test

import (
	"context"
	"sort"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminListConnections(t *testing.T) {
	t.Run("Listing the connections with invalid params fails", testAdminListConnectionsWithInvalidParamsFails)
	t.Run("Listing the connections with valid params succeeds", testAdminListConnectionsWithValidParamsSucceeds)
}

func testAdminListConnectionsWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty network",
			params: api.AdminListConnectionsParams{
				Network: "",
			},
			expectedError: api.ErrNetworkIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newListConnectionsHandler(tt)

			// when
			response, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, response)
		})
	}
}

func testAdminListConnectionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	network := vgrand.RandomStr(5)
	url := "http://" + vgrand.RandomStr(5)
	hostname := vgrand.RandomStr(5)
	otherHostname := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)
	otherWallet, _ := walletWithKey(t)

	// setup
	handler := newListConnectionsHandler(t)
	sessions := session.NewSessions()
	if _, err := sessions.ConnectWallet(hostname, expectedWallet); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.ConnectWallet(otherHostname, expectedWallet); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.ConnectWallet(hostname, otherWallet); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.ConnectWallet(otherHostname, otherWallet); err != nil {
		t.Fatal(err)
	}
	if err := handler.servicesManager.RegisterService(network, url, sessions, dummyServiceShutdownSwitch()); err != nil {
		t.Fatal(err)
	}

	// when
	response, errorDetails := handler.handle(t, ctx, api.AdminListConnectionsParams{
		Network: network,
	})

	// then
	require.Nil(t, errorDetails)
	expectedConnections := []session.Connection{
		{Hostname: hostname, Wallet: expectedWallet.Name()},
		{Hostname: otherHostname, Wallet: expectedWallet.Name()},
		{Hostname: hostname, Wallet: otherWallet.Name()},
		{Hostname: otherHostname, Wallet: otherWallet.Name()},
	}
	sort.SliceStable(expectedConnections, func(i, j int) bool {
		if expectedConnections[i].Hostname == expectedConnections[j].Hostname {
			return expectedConnections[i].Wallet < expectedConnections[j].Wallet
		}
		return expectedConnections[i].Hostname < expectedConnections[j].Hostname
	})
	assert.Equal(t, response.ActiveConnections, expectedConnections)
}

type adminListConnectionsHandler struct {
	*api.AdminListConnections
	ctrl            *gomock.Controller
	servicesManager *api.ServicesManager
	walletStore     *mocks.MockWalletStore
	tokenStore      *mocks.MockTokenStore
}

func (h *adminListConnectionsHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminListConnectionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminListConnectionsResult)
		if !ok {
			t.Fatal("AdminIsolateKey handler result is not a api.AdminListConnectionsResult")
		}
		return result, err
	}
	return api.AdminListConnectionsResult{}, err
}

func newListConnectionsHandler(t *testing.T) *adminListConnectionsHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	walletStore := mocks.NewMockWalletStore(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	tokenStore.EXPECT().ListTokens().AnyTimes().Return([]session.TokenSummary{}, nil)
	servicesManager := api.NewServicesManager(tokenStore, walletStore)

	return &adminListConnectionsHandler{
		AdminListConnections: api.NewAdminListConnections(servicesManager),
		ctrl:                 ctrl,
		servicesManager:      servicesManager,
		walletStore:          walletStore,
		tokenStore:           tokenStore,
	}
}
