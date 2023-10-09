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
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/network"
)

type AdminListNetworksResult struct {
	Networks []AdminListNetworkResult `json:"networks"`
}

type AdminListNetworkResult struct {
	Name     string             `json:"name"`
	Metadata []network.Metadata `json:"metadata"`
}

type AdminListNetworks struct {
	networkStore NetworkStore
}

// Handle List all registered networks.
func (h *AdminListNetworks) Handle(_ context.Context, _ jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	networks, err := h.networkStore.ListNetworks()
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not list the networks: %w", err))
	}

	netsWithMetadata := make([]AdminListNetworkResult, 0, len(networks))
	for _, networkName := range networks {
		net, err := h.networkStore.GetNetwork(networkName)
		if err != nil {
			continue
		}
		netsWithMetadata = append(netsWithMetadata, AdminListNetworkResult{
			Name:     networkName,
			Metadata: net.Metadata,
		})
	}

	return AdminListNetworksResult{
		Networks: netsWithMetadata,
	}, nil
}

func NewAdminListNetworks(
	networkStore NetworkStore,
) *AdminListNetworks {
	return &AdminListNetworks{
		networkStore: networkStore,
	}
}
