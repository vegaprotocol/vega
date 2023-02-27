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

func TestAdminRenameNetwork(t *testing.T) {
	t.Run("Renaming a network with invalid params fails", testRenamingNetworkWithInvalidParamsFails)
	t.Run("Renaming a network with valid params succeeds", testRenamingNetworkWithValidParamsSucceeds)
	t.Run("Renaming a network that does not exists fails", testRenamingNetworkThatDoesNotExistsFails)
	t.Run("Getting internal error during existing network verification does not rename the network", testGettingInternalErrorDuringExistingNetworkVerificationDoesNotRenameNetwork)
	t.Run("Renaming a network that with name that is already taken fails", testRenamingNetworkWithNameAlreadyTakenFails)
	t.Run("Getting internal error during non-existing network verification does not rename the network", testGettingInternalErrorDuringNonExistingNetworkVerificationDoesNotRenameNetwork)
	t.Run("Getting internal error during renaming does not rename the network", testGettingInternalErrorDuringRenamingDoesNotRenameNetwork)
}

func testRenamingNetworkWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminRenameNetworkParams{
				Network: "",
				NewName: vgrand.RandomStr(5),
			},
			expectedError: api.ErrNetworkIsRequired,
		}, {
			name: "with empty new name",
			params: api.AdminRenameNetworkParams{
				Network: vgrand.RandomStr(5),
				NewName: "",
			},
			expectedError: api.ErrNewNameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newRenameNetworkHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testRenamingNetworkWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().NetworkExists(newName).Times(1).Return(false, nil)
	handler.networkStore.EXPECT().RenameNetwork(name, newName).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameNetworkParams{
		Network: name,
		NewName: newName,
	})

	// then
	require.Nil(t, errorDetails)
}

func testRenamingNetworkThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameNetworkParams{
		Network: name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrNetworkDoesNotExist)
}

func testGettingInternalErrorDuringExistingNetworkVerificationDoesNotRenameNetwork(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameNetworkParams{
		Network: name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the network existence: %w", assert.AnError))
}

func testRenamingNetworkWithNameAlreadyTakenFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().NetworkExists(newName).Times(1).Return(true, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameNetworkParams{
		Network: name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrNetworkAlreadyExists)
}

func testGettingInternalErrorDuringNonExistingNetworkVerificationDoesNotRenameNetwork(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().NetworkExists(newName).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameNetworkParams{
		Network: name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the network existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringRenamingDoesNotRenameNetwork(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)
	newName := vgrand.RandomStr(5)

	// setup
	handler := newRenameNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().NetworkExists(newName).Times(1).Return(false, nil)
	handler.networkStore.EXPECT().RenameNetwork(name, newName).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminRenameNetworkParams{
		Network: name,
		NewName: newName,
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not rename the network: %w", assert.AnError))
}

type renameNetworkHandler struct {
	*api.AdminRenameNetwork
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *renameNetworkHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Nil(t, rawResult)
	return err
}

func newRenameNetworkHandler(t *testing.T) *renameNetworkHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &renameNetworkHandler{
		AdminRenameNetwork: api.NewAdminRenameNetwork(networkStore),
		ctrl:               ctrl,
		networkStore:       networkStore,
	}
}
