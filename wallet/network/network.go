package network

import (
	"errors"

	"code.vegaprotocol.io/vega/wallet/service/encoding"
)

var ErrNetworkDoesNotHaveGRPCHostConfigured = errors.New("network configuration does not have any gRPC host set")

type Network struct {
	Name        string            `json:"name"`
	Level       encoding.LogLevel `json:"level"`
	TokenExpiry encoding.Duration `json:"tokenExpiry"`
	Port        int               `json:"port"`
	Host        string            `json:"host"`
	API         APIConfig         `json:"api"`
}

type APIConfig struct {
	GRPC    GRPCConfig    `json:"grpc"`
	REST    RESTConfig    `json:"rest"`
	GraphQL GraphQLConfig `json:"graphQl"`
}

type GRPCConfig struct {
	Hosts   []string `json:"hosts"`
	Retries uint64   `json:"retries"`
}

type RESTConfig struct {
	Hosts []string `json:"hosts"`
}

type GraphQLConfig struct {
	Hosts []string `json:"hosts"`
}

func (n *Network) EnsureCanConnectGRPCNode() error {
	if len(n.API.GRPC.Hosts) > 0 && len(n.API.GRPC.Hosts[0]) > 0 {
		return nil
	}
	return ErrNetworkDoesNotHaveGRPCHostConfigured
}
