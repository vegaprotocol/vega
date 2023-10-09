// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	t.Run("Documentation matches the code", testAdminCloseConnectionSchemaCorrect)
	t.Run("Closing a connection with invalid params fails", testAdminCloseConnectionWithInvalidParamsFails)
	t.Run("Closing a connection with valid params succeeds", testAdminCloseConnectionWithValidParamsSucceeds)
}

func testAdminCloseConnectionSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.close_connection", api.AdminCloseConnectionParams{}, nil)
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
