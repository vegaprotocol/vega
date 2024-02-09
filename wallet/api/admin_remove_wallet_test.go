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
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminRemoveWallet(t *testing.T) {
	t.Run("Documentation matches the code", testAdminRemoveWalletSchemaCorrect)
	t.Run("Removing a wallet with invalid params fails", testRemovingWalletWithInvalidParamsFails)
	t.Run("Removing a wallet with valid params succeeds", testRemovingWalletWithValidParamsSucceeds)
	t.Run("Removing a wallet that does not exists fails", testRemovingWalletThatDoesNotExistsFails)
	t.Run("Getting internal error during verification does not remove the wallet", testGettingInternalErrorDuringVerificationDoesNotRemoveWallet)
}

func testAdminRemoveWalletSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.remove_wallet", api.AdminRemoveWalletParams{}, nil)
}

func testRemovingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminRemoveWalletParams{
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
			handler := newRemoveWalletHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testRemovingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newRemoveWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(true, nil)
	handler.walletStore.EXPECT().DeleteWallet(ctx, name).Times(1).Return(nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRemoveWalletParams{
		Wallet: name,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Nil(t, result)
}

func testRemovingWalletThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newRemoveWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRemoveWalletParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrWalletDoesNotExist)
}

func testGettingInternalErrorDuringVerificationDoesNotRemoveWallet(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newRemoveWalletHandler(t)
	// -- expected calls
	handler.walletStore.EXPECT().WalletExists(ctx, name).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminRemoveWalletParams{
		Wallet: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the wallet exists: %w", assert.AnError))
}

type removeWalletHandler struct {
	*api.AdminRemoveWallet
	ctrl        *gomock.Controller
	walletStore *mocks.MockWalletStore
}

func (h *removeWalletHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	t.Helper()

	return h.Handle(ctx, params)
}

func newRemoveWalletHandler(t *testing.T) *removeWalletHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	walletStore := mocks.NewMockWalletStore(ctrl)

	return &removeWalletHandler{
		AdminRemoveWallet: api.NewAdminRemoveWallet(walletStore),
		ctrl:              ctrl,
		walletStore:       walletStore,
	}
}
