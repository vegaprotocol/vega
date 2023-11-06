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

func TestAdminCloseConnectionsToWallet(t *testing.T) {
	t.Run("Documentation matches the code", testAdminCloseConnectionsToWalletSchemaCorrect)
	t.Run("Closing a connection with invalid params fails", testAdminCloseConnectionsToWalletWithInvalidParamsFails)
	t.Run("Closing a connection with valid params succeeds", testAdminCloseConnectionsToWalletWithValidParamsSucceeds)
}

func testAdminCloseConnectionsToWalletSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.close_connections_to_wallet", api.AdminCloseConnectionsToWalletParams{}, nil)
}

func testAdminCloseConnectionsToWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminCloseConnectionsToWalletParams{
				Wallet: "",
			},
			expectedError: api.ErrWalletIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newCloseConnectionsToWalletHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminCloseConnectionsToWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname1 := vgrand.RandomStr(5)
	hostname2 := vgrand.RandomStr(5)
	wallet1 := vgrand.RandomStr(5)
	wallet2 := vgrand.RandomStr(5)
	wallet3 := vgrand.RandomStr(5)

	// setup
	handler := newCloseConnectionsToWalletHandler(t)
	// -- expected calls
	handler.connectionsManager.EXPECT().ListSessionConnections().Times(1).Return([]api.Connection{
		{
			Hostname: hostname1,
			Wallet:   wallet1,
		}, {
			Hostname: hostname1,
			Wallet:   wallet2,
		}, {
			Hostname: hostname1,
			Wallet:   wallet3,
		}, {
			Hostname: hostname2,
			Wallet:   wallet1,
		}, {
			Hostname: hostname2,
			Wallet:   wallet2,
		}, {
			Hostname: hostname2,
			Wallet:   wallet3,
		},
	})
	handler.connectionsManager.EXPECT().EndSessionConnection(hostname1, wallet1).Times(1)
	handler.connectionsManager.EXPECT().EndSessionConnection(hostname2, wallet1).Times(1)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionsToWalletParams{
		Wallet: wallet1,
	})

	// then
	require.Nil(t, errorDetails)
}

type adminCloseConnectionsToWalletHandler struct {
	*api.AdminCloseConnectionsToWallet
	ctrl               *gomock.Controller
	connectionsManager *mocks.MockConnectionsManager
	walletStore        *mocks.MockWalletStore
}

func (h *adminCloseConnectionsToWalletHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Empty(t, rawResult)
	return err
}

func newCloseConnectionsToWalletHandler(t *testing.T) *adminCloseConnectionsToWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	walletStore := mocks.NewMockWalletStore(ctrl)
	connectionsManager := mocks.NewMockConnectionsManager(ctrl)

	return &adminCloseConnectionsToWalletHandler{
		AdminCloseConnectionsToWallet: api.NewAdminCloseConnectionsToWallet(connectionsManager),
		ctrl:                          ctrl,
		connectionsManager:            connectionsManager,
		walletStore:                   walletStore,
	}
}
