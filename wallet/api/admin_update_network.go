package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/network"
	"github.com/mitchellh/mapstructure"
)

type AdminUpdateNetworkParams struct {
	Name string `json:"name"`
	API  struct {
		GRPCConfig struct {
			Hosts   []string `json:"hosts"`
			Retries uint64   `json:"retries"`
		} `json:"grpcConfig"`
		RESTConfig struct {
			Hosts []string `json:"hosts"`
		} `json:"restConfig"`
		GraphQLConfig struct {
			Hosts []string `json:"hosts"`
		} `json:"graphQLConfig"`
	} `json:"api"`
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

	return network.Network{
		Name: params.Name,
		API: network.APIConfig{
			GRPC: network.GRPCConfig{
				Hosts:   params.API.GRPCConfig.Hosts,
				Retries: params.API.GRPCConfig.Retries,
			},
			REST: network.RESTConfig{
				Hosts: params.API.RESTConfig.Hosts,
			},
			GraphQL: network.GraphQLConfig{
				Hosts: params.API.GraphQLConfig.Hosts,
			},
		},
	}, nil
}

func NewAdminUpdateNetwork(
	networkStore NetworkStore,
) *AdminUpdateNetwork {
	return &AdminUpdateNetwork{
		networkStore: networkStore,
	}
}
