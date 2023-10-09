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
	GRPC    HostConfig `json:"grpc"`
	REST    HostConfig `json:"rest"`
	GraphQL HostConfig `json:"graphQL"`
}

type HostConfig struct {
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
