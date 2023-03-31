package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestAdminCloseConnection(t *testing.T) {
	t.Run("Closing a connection with invalid params fails", testAdminCloseConnectionWithInvalidParamsFails)
	t.Run("Closing a connection with valid params succeeds", testAdminCloseConnectionWithValidParamsSucceeds)
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
				Hostname: vgrand.RandomStr(5),
				Wallet:   "",
			},
			expectedError: api.ErrWalletIsRequired,
		}, {
			name: "with empty hostname",
			params: api.AdminCloseConnectionParams{
				Wallet:   vgrand.RandomStr(5),
				Hostname: "",
			},
			expectedError: api.ErrHostnameIsRequired,
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
	hostname := vgrand.RandomStr(5)
	wallet := vgrand.RandomStr(5)

	// setup
	handler := newCloseConnectionHandler(t)
	// -- expected calls
	handler.connectionsManager.EXPECT().EndSessionConnection(hostname, wallet).Times(1)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionParams{
		Hostname: hostname,
		Wallet:   wallet,
	})

	// then
	require.Nil(t, errorDetails)
}

type adminCloseConnectionHandler struct {
	*api.AdminCloseConnection
	ctrl               *gomock.Controller
	walletStore        *mocks.MockWalletStore
	connectionsManager *mocks.MockConnectionsManager
}

func (h *adminCloseConnectionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Empty(t, rawResult)
	return err
}

func newCloseConnectionHandler(t *testing.T) *adminCloseConnectionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	walletStore := mocks.NewMockWalletStore(ctrl)
	connectionsManager := mocks.NewMockConnectionsManager(ctrl)

	return &adminCloseConnectionHandler{
		AdminCloseConnection: api.NewAdminCloseConnection(connectionsManager),
		ctrl:                 ctrl,
		connectionsManager:   connectionsManager,
		walletStore:          walletStore,
	}
}
