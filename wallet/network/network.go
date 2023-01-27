package network

import (
	"errors"
	"fmt"
)

var ErrNetworkDoesNotHaveGRPCHostConfigured = errors.New("network configuration does not have any gRPC host set")

type Network struct {
	Name string    `json:"name"`
	API  APIConfig `json:"api"`
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

func GetNetwork(store Store, name string) (*Network, error) {
	exists, err := store.NetworkExists(name)
	if err != nil {
		return nil, fmt.Errorf("couldn't verify network existence: %w", err)
	}
	if !exists {
		return nil, NewDoesNotExistError(name)
	}
	n, err := store.GetNetwork(name)
	if err != nil {
		return nil, fmt.Errorf("couldn't get network %s: %w", name, err)
	}

	return n, nil
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/store_mock.go -package mocks code.vegaprotocol.io/vega/wallet/network Store
type Store interface {
	NetworkExists(string) (bool, error)
	GetNetwork(string) (*Network, error)
}
