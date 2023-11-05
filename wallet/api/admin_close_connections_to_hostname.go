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

	"github.com/mitchellh/mapstructure"
)

type AdminCloseConnectionsToHostnameParams struct {
	Hostname string `json:"hostname"`
}

type AdminCloseConnectionsToHostname struct {
	connectionsManager ConnectionsManager
}

// Handle closes all the connections from the specified hostname to any wallet
// opened in the service that run against the specified network.
// It does not fail if the service or the connections are already closed.
func (h *AdminCloseConnectionsToHostname) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminCloseConnectionsToHostnameParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	connections := h.connectionsManager.ListSessionConnections()

	for _, connection := range connections {
		if connection.Hostname == params.Hostname {
			h.connectionsManager.EndSessionConnection(params.Hostname, connection.Wallet)
		}
	}

	return nil, nil
}

func validateAdminCloseConnectionsToHostnameParams(rawParams jsonrpc.Params) (AdminCloseConnectionsToHostnameParams, error) {
	if rawParams == nil {
		return AdminCloseConnectionsToHostnameParams{}, ErrParamsRequired
	}

	params := AdminCloseConnectionsToHostnameParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCloseConnectionsToHostnameParams{}, ErrParamsDoNotMatch
	}

	if params.Hostname == "" {
		return AdminCloseConnectionsToHostnameParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewAdminCloseConnectionsToHostname(connectionsManager ConnectionsManager) *AdminCloseConnectionsToHostname {
	return &AdminCloseConnectionsToHostname{
		connectionsManager: connectionsManager,
	}
}
