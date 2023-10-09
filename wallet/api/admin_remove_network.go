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
	"github.com/mitchellh/mapstructure"
)

type AdminRemoveNetworkParams struct {
	Name string `json:"name"`
}

type AdminRemoveNetwork struct {
	networkStore NetworkStore
}

// Handle removes a wallet from the computer.
func (h *AdminRemoveNetwork) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRemoveNetworkParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.networkStore.NetworkExists(params.Name); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if !exist {
		return nil, InvalidParams(ErrNetworkDoesNotExist)
	}

	if err := h.networkStore.DeleteNetwork(params.Name); err != nil {
		return nil, InternalError(fmt.Errorf("could not remove the wallet: %w", err))
	}

	return nil, nil
}

func validateRemoveNetworkParams(rawParams jsonrpc.Params) (AdminRemoveNetworkParams, error) {
	if rawParams == nil {
		return AdminRemoveNetworkParams{}, ErrParamsRequired
	}

	params := AdminRemoveNetworkParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRemoveNetworkParams{}, ErrParamsDoNotMatch
	}

	if params.Name == "" {
		return AdminRemoveNetworkParams{}, ErrNetworkNameIsRequired
	}

	return params, nil
}

func NewAdminRemoveNetwork(
	networkStore NetworkStore,
) *AdminRemoveNetwork {
	return &AdminRemoveNetwork{
		networkStore: networkStore,
	}
}
