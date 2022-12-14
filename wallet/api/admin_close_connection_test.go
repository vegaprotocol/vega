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

func TestAdminCloseConnection(t *testing.T) {
	t.Run("Closing a connection with invalid params fails", testAdminCloseConnectionWithInvalidParamsFails)
	t.Run("Closing a connection with valid params succeeds", testAdminCloseConnectionWithValidParamsSucceeds)
	t.Run("Closing a connection on unknown network doesn't fail", testAdminCloseConnectionOnUnknownNetworkDoesNotFail)
	t.Run("Closing a connection on unknown hostname doesn't fail", testAdminCloseConnectionOnUnknownHostnameDoesNotFail)
	t.Run("Closing a connection on unknown wallet doesn't fail", testAdminCloseConnectionOnUnknownWalletDoesNotFail)
}

func testAdminCloseConnectionWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty wallet",
			params: api.AdminCloseConnectionParams{
				Network:  vgrand.RandomStr(5),
				Hostname: vgrand.RandomStr(5),
				Wallet:   "",
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty hostname",
			params: api.AdminCloseConnectionParams{
				Network:  vgrand.RandomStr(5),
				Wallet:   vgrand.RandomStr(5),
				Hostname: "",
			},
			expectedError: api.ErrHostnameIsRequired,
		}, {
			name: "with empty network",
			params: api.AdminCloseConnectionParams{
				Hostname: vgrand.RandomStr(5),
				Wallet:   vgrand.RandomStr(5),
				Network:  "",
			},
			expectedError: api.ErrNetworkIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newCloseConnectionHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminCloseConnectionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	network := vgrand.RandomStr(5)
	url := "http://" + vgrand.RandomStr(5)
	hostname := vgrand.RandomStr(5)
	otherHostname := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)
	otherWallet, _ := walletWithKey(t)

	// setup
	handler := newCloseConnectionHandler(t)
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
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionParams{
		Network:  network,
		Hostname: hostname,
		Wallet:   expectedWallet.Name(),
	})

	// then
	require.Nil(t, errorDetails)
	assert.NotContains(t, sessions.ListConnections(), session.Connection{
		Hostname: hostname,
		Wallet:   expectedWallet.Name(),
	})
}

func testAdminCloseConnectionOnUnknownNetworkDoesNotFail(t *testing.T) {
	// given
	ctx := context.Background()
	network := vgrand.RandomStr(5)
	url := "http://" + vgrand.RandomStr(5)
	hostname := vgrand.RandomStr(5)
	otherHostname := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)
	otherWallet, _ := walletWithKey(t)

	// setup
	handler := newCloseConnectionHandler(t)
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
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionParams{
		Network:  network,
		Hostname: vgrand.RandomStr(5),
		Wallet:   expectedWallet.Name(),
	})

	// then
	require.Nil(t, errorDetails)
	connections := sessions.ListConnections()
	assert.Len(t, connections, 4)
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
	assert.Equal(t, connections, expectedConnections)
}

func testAdminCloseConnectionOnUnknownHostnameDoesNotFail(t *testing.T) {
	// given
	ctx := context.Background()
	network := vgrand.RandomStr(5)
	url := "http://" + vgrand.RandomStr(5)
	hostname := vgrand.RandomStr(5)
	otherHostname := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)
	otherWallet, _ := walletWithKey(t)

	// setup
	handler := newCloseConnectionHandler(t)
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
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionParams{
		Network:  network,
		Hostname: vgrand.RandomStr(5),
		Wallet:   expectedWallet.Name(),
	})

	// then
	require.Nil(t, errorDetails)
	connections := sessions.ListConnections()
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
	assert.Equal(t, connections, expectedConnections)
}

func testAdminCloseConnectionOnUnknownWalletDoesNotFail(t *testing.T) {
	// given
	ctx := context.Background()
	network := vgrand.RandomStr(5)
	url := "http://" + vgrand.RandomStr(5)
	hostname := vgrand.RandomStr(5)
	otherHostname := vgrand.RandomStr(5)
	expectedWallet, _ := walletWithKey(t)
	otherWallet, _ := walletWithKey(t)

	// setup
	handler := newCloseConnectionHandler(t)
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
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionParams{
		Network:  network,
		Hostname: hostname,
		Wallet:   vgrand.RandomStr(5),
	})

	// then
	require.Nil(t, errorDetails)
	connections := sessions.ListConnections()
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
	assert.Equal(t, connections, expectedConnections)
}

type adminCloseConnectionHandler struct {
	*api.AdminCloseConnection
	ctrl            *gomock.Controller
	servicesManager *api.ServicesManager
	walletStore     *mocks.MockWalletStore
	tokenStore      *mocks.MockTokenStore
}

func (h *adminCloseConnectionHandler) handle(t *testing.T, ctx context.Context, params interface{}) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	require.Empty(t, rawResult)
	return err
}

func newCloseConnectionHandler(t *testing.T) *adminCloseConnectionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	walletStore := mocks.NewMockWalletStore(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	tokenStore.EXPECT().ListTokens().AnyTimes().Return([]session.TokenSummary{}, nil)
	servicesManager := api.NewServicesManager(tokenStore, walletStore)

	return &adminCloseConnectionHandler{
		AdminCloseConnection: api.NewAdminCloseConnection(servicesManager),
		ctrl:                 ctrl,
		servicesManager:      servicesManager,
		walletStore:          walletStore,
		tokenStore:           tokenStore,
	}
}
