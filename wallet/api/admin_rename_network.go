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

type AdminRenameNetworkParams struct {
	Network string `json:"network"`
	NewName string `json:"newName"`
}

type AdminRenameNetwork struct {
	networkStore NetworkStore
}

// Handle renames a network.
func (h *AdminRenameNetwork) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRenameNetworkParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.networkStore.NetworkExists(params.Network); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if !exist {
		return nil, InvalidParams(ErrNetworkDoesNotExist)
	}

	if exist, err := h.networkStore.NetworkExists(params.NewName); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if exist {
		return nil, InvalidParams(ErrNetworkAlreadyExists)
	}

	if err := h.networkStore.RenameNetwork(params.Network, params.NewName); err != nil {
		return nil, InternalError(fmt.Errorf("could not rename the network: %w", err))
	}

	return nil, nil
}

func validateRenameNetworkParams(rawParams jsonrpc.Params) (AdminRenameNetworkParams, error) {
	if rawParams == nil {
		return AdminRenameNetworkParams{}, ErrParamsRequired
	}

	params := AdminRenameNetworkParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRenameNetworkParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" {
		return AdminRenameNetworkParams{}, ErrNetworkIsRequired
	}

	if params.NewName == "" {
		return AdminRenameNetworkParams{}, ErrNewNameIsRequired
	}

	return params, nil
}

func NewAdminRenameNetwork(
	networkStore NetworkStore,
) *AdminRenameNetwork {
	return &AdminRenameNetwork{
		networkStore: networkStore,
	}
}
