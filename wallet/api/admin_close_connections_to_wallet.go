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

type AdminCloseConnectionsToWalletParams struct {
	Wallet string `json:"wallet"`
}

type AdminCloseConnectionsToWallet struct {
	connectionsManager ConnectionsManager
}

// Handle closes all the connections from any hostname to the specified wallet
// opened in the service that run against the specified network.
// It does not fail if the service or the connections are already closed.
func (h *AdminCloseConnectionsToWallet) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminCloseConnectionsToWalletParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	connections := h.connectionsManager.ListSessionConnections()

	for _, connection := range connections {
		if connection.Wallet == params.Wallet {
			h.connectionsManager.EndSessionConnection(connection.Hostname, params.Wallet)
		}
	}

	return nil, nil
}

func validateAdminCloseConnectionsToWalletParams(rawParams jsonrpc.Params) (AdminCloseConnectionsToWalletParams, error) {
	if rawParams == nil {
		return AdminCloseConnectionsToWalletParams{}, ErrParamsRequired
	}

	params := AdminCloseConnectionsToWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCloseConnectionsToWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminCloseConnectionsToWalletParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminCloseConnectionsToWallet(connectionsManager ConnectionsManager) *AdminCloseConnectionsToWallet {
	return &AdminCloseConnectionsToWallet{
		connectionsManager: connectionsManager,
	}
}
