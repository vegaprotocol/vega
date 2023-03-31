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
		return nil, internalError(fmt.Errorf("could not list the networks: %w", err))
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
