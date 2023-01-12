package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
)

type AdminListConnectionsResult struct {
	ActiveConnections []Connection `json:"activeConnections"`
}

type AdminListConnections struct {
	connectionsManager ConnectionsManager
}

func (h *AdminListConnections) Handle(_ context.Context, _ jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	return AdminListConnectionsResult{
		ActiveConnections: h.connectionsManager.ListSessionConnections(),
	}, nil
}

func NewAdminListConnections(connectionsManager ConnectionsManager) *AdminListConnections {
	return &AdminListConnections{
		connectionsManager: connectionsManager,
	}
}
