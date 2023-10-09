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
