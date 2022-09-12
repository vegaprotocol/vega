package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminRemoveNetworkParams struct {
	Network string `json:"network"`
}

type AdminRemoveNetwork struct {
	networkStore NetworkStore
}

// Handle removes a wallet from the computer.
func (h *AdminRemoveNetwork) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRemoveNetworkParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.networkStore.NetworkExists(params.Network); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrNetworkDoesNotExist)
	}

	if err := h.networkStore.DeleteNetwork(params.Network); err != nil {
		return nil, internalError(fmt.Errorf("could not remove the wallet: %w", err))
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

	if params.Network == "" {
		return AdminRemoveNetworkParams{}, ErrNetworkIsRequired
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
