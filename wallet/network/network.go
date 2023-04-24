package network

import (
	"errors"
	"fmt"
)

var ErrNetworkDoesNotHaveGRPCHostConfigured = errors.New("network configuration does not have any gRPC host set")

type Network struct {
	Name     string     `json:"name"`
	Metadata []Metadata `json:"metadata"`
	API      APIConfig  `json:"api"`
	Apps     AppsConfig `json:"apps"`
}

type APIConfig struct {
	GRPC    GRPCConfig    `json:"grpc"`
	REST    RESTConfig    `json:"rest"`
	GraphQL GraphQLConfig `json:"graphQL"`
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

type AppsConfig struct {
	Console    string `json:"console"`
	Governance string `json:"governance"`
	Explorer   string `json:"explorer"`
}

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
		return nil, fmt.Errorf("couldn't verify network exists: %w", err)
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
