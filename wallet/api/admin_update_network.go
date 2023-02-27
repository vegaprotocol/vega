package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/network"
	"github.com/mitchellh/mapstructure"
)

type AdminUpdateNetworkParams struct {
	Name     string             `json:"name"`
	Metadata []network.Metadata `json:"metadata"`
	API      network.APIConfig  `json:"api"`
	Apps     network.AppsConfig `json:"apps"`
}
type AdminUpdateNetwork struct {
	networkStore NetworkStore
}

func (h *AdminUpdateNetwork) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	updatedNetwork, err := validateUpdateNetworkParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exists, err := h.networkStore.NetworkExists(updatedNetwork.Name); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if !exists {
		return nil, invalidParams(ErrNetworkDoesNotExist)
	}

	if err := h.networkStore.SaveNetwork(&updatedNetwork); err != nil {
		return nil, internalError(fmt.Errorf("could not save the network: %w", err))
	}
	return nil, nil
}

func validateUpdateNetworkParams(rawParams jsonrpc.Params) (network.Network, error) {
	if rawParams == nil {
		return network.Network{}, ErrParamsRequired
	}

	params := AdminUpdateNetworkParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return network.Network{}, ErrParamsDoNotMatch
	}

	if params.Name == "" {
		return network.Network{}, ErrNetworkNameIsRequired
	}

	net := network.Network{
		Name:     params.Name,
		Metadata: params.Metadata,
		API:      params.API,
		Apps:     params.Apps,
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
