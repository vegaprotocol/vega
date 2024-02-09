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

	"github.com/mitchellh/mapstructure"
)

type AdminUpdateNetwork struct {
	networkStore NetworkStore
}

func (h *AdminUpdateNetwork) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	updatedNetwork, err := validateUpdateNetworkParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exists, err := h.networkStore.NetworkExists(updatedNetwork.Name); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if !exists {
		return nil, InvalidParams(ErrNetworkDoesNotExist)
	}

	if err := h.networkStore.SaveNetwork(&updatedNetwork); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the network: %w", err))
	}
	return nil, nil
}

func validateUpdateNetworkParams(rawParams jsonrpc.Params) (network.Network, error) {
	if rawParams == nil {
		return network.Network{}, ErrParamsRequired
	}

	params := AdminNetwork{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return network.Network{}, ErrParamsDoNotMatch
	}

	if params.Name == "" {
		return network.Network{}, ErrNetworkNameIsRequired
	}

	net := network.Network{
		Name:     params.Name,
		Metadata: params.Metadata,
		API: network.APIConfig{
			GRPC: network.HostConfig{
				Hosts: params.API.GRPC.Hosts,
			},
			REST: network.HostConfig{
				Hosts: params.API.REST.Hosts,
			},
			GraphQL: network.HostConfig{
				Hosts: params.API.GraphQL.Hosts,
			},
		},
		Apps: network.AppsConfig{
			Console:    params.Apps.Console,
			Governance: params.Apps.Governance,
			Explorer:   params.Apps.Explorer,
		},
	}

	if err := net.EnsureCanConnectGRPCNode(); err != nil {
		return network.Network{}, err
	}

	return net, nil
}

func NewAdminUpdateNetwork(
	networkStore NetworkStore,
) *AdminUpdateNetwork {
	return &AdminUpdateNetwork{
		networkStore: networkStore,
	}
}
