package api

import (
	"context"
	"fmt"

	vgencoding "code.vegaprotocol.io/vega/libs/encoding"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminDescribeNetworkParams struct {
	Name string `json:"name"`
}

type AdminDescribeNetworkResult struct {
	Name        string              `json:"name"`
	LogLevel    vgencoding.LogLevel `json:"logLevel"`
	TokenExpiry vgencoding.Duration `json:"tokenExpiry"`
	Port        int                 `json:"port"`
	Host        string              `json:"host"`
	API         struct {
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
		return nil, internalError(fmt.Errorf("could not retrieve the network: %w", err))
	}

	resp := AdminDescribeNetworkResult{
		Name:        n.Name,
		LogLevel:    n.LogLevel,
		TokenExpiry: n.TokenExpiry,
		Port:        n.Port,
		Host:        n.Host,
	}

	resp.API.GRPCConfig.Hosts = n.API.GRPC.Hosts
	resp.API.GRPCConfig.Retries = n.API.GRPC.Retries
	resp.API.RESTConfig.Hosts = n.API.REST.Hosts
	resp.API.GraphQLConfig.Hosts = n.API.GraphQL.Hosts
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
