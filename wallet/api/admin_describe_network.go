package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminDescribeNetworkParams struct {
	Name string `json:"name"`
}

type AdminDescribeNetwork struct {
	networkStore NetworkStore
}

func (h *AdminDescribeNetwork) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateDescribeNetworkParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.networkStore.NetworkExists(params.Name); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the network existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrNetworkDoesNotExist)
	}

	n, err := h.networkStore.GetNetwork(params.Name)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the network configuration: %w", err))
	}

	resp := AdminNetwork{
		Name:     n.Name,
		Metadata: n.Metadata,
		API: AdminAPIConfig{
			GRPC: AdminGRPCConfig{
				Hosts:   n.API.GRPC.Hosts,
				Retries: n.API.GRPC.Retries,
			},
			REST: AdminRESTConfig{
				Hosts: n.API.REST.Hosts,
			},
			GraphQL: AdminGraphQLConfig{
				Hosts: n.API.GraphQL.Hosts,
			},
		},
		Apps: AdminAppConfig{
			Explorer:   n.Apps.Explorer,
			Console:    n.Apps.Console,
			Governance: n.Apps.Governance,
		},
	}

	// make sure nil maps come through as empty slices
	if resp.API.GRPC.Hosts == nil {
		resp.API.GRPC.Hosts = []string{}
	}
	if resp.API.GraphQL.Hosts == nil {
		resp.API.GraphQL.Hosts = []string{}
	}
	if resp.API.REST.Hosts == nil {
		resp.API.REST.Hosts = []string{}
	}

	return resp, nil
}

func validateDescribeNetworkParams(rawParams jsonrpc.Params) (AdminDescribeNetworkParams, error) {
	if rawParams == nil {
		return AdminDescribeNetworkParams{}, ErrParamsRequired
	}

	params := AdminDescribeNetworkParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribeNetworkParams{}, ErrParamsDoNotMatch
	}

	if params.Name == "" {
		return AdminDescribeNetworkParams{}, ErrNetworkNameIsRequired
	}

	return params, nil
}

func NewAdminDescribeNetwork(
	networkStore NetworkStore,
) *AdminDescribeNetwork {
	return &AdminDescribeNetwork{
		networkStore: networkStore,
	}
}
