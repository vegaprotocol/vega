package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminListConnections(t *testing.T) {
	t.Run("Documentation matches the code", testAdminListConnectionsSchemaCorrect)
	t.Run("Listing the connections succeeds", testAdminListConnectionsSucceeds)
}

func testAdminListConnectionsSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.list_connections", nil, api.AdminListConnectionsResult{})
}

func testAdminListConnectionsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname1 := vgrand.RandomStr(5)
	hostname2 := vgrand.RandomStr(5)
	wallet1 := vgrand.RandomStr(5)
	wallet2 := vgrand.RandomStr(5)
	wallet3 := vgrand.RandomStr(5)
	list := []api.Connection{
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
	}

	// setup
	handler := newListConnectionsHandler(t)
	// -- expected calls
	handler.connectionsManager.EXPECT().ListSessionConnections().Times(1).Return(list)

	// when
	response, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, list, response.ActiveConnections)
}

type adminListConnectionsHandler struct {
	*api.AdminListConnections
	ctrl               *gomock.Controller
	connectionsManager *mocks.MockConnectionsManager
}

func (h *adminListConnectionsHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminListConnectionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
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

	connectionsManager := mocks.NewMockConnectionsManager(ctrl)

	return &adminListConnectionsHandler{
		AdminListConnections: api.NewAdminListConnections(connectionsManager),
		ctrl:                 ctrl,
		connectionsManager:   connectionsManager,
	}
}
