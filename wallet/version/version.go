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

package version

import (
	"errors"
	"sort"
	"strings"

	vgversion "code.vegaprotocol.io/vega/libs/version"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/network"
)

var ErrCouldNotListNetworks = errors.New("couldn't list the networks")

// RequestVersionFn is the function in charge of retrieving the network version
// ran by the host lists.
type RequestVersionFn func(hosts []string) (string, error)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/store_mock.go -package mocks code.vegaprotocol.io/vega/wallet/version NetworkStore
type NetworkStore interface {
	ListNetworks() ([]string, error)
	GetNetwork(string) (*network.Network, error)
}

func IsUnreleased() bool {
	return vgversion.IsUnreleased(coreversion.Get())
}

type GetSoftwareVersionResponse struct {
	Version string `json:"version"`
	GitHash string `json:"gitHash"`
}

func GetSoftwareVersionInfo() *GetSoftwareVersionResponse {
	response := &GetSoftwareVersionResponse{
		Version: coreversion.Get(),
		GitHash: coreversion.GetCommitHash(),
	}

	return response
}

type CheckSoftwareCompatibilityResponse struct {
	NetworksCompatibility []NetworkCompatibility `json:"networksCompatibility"`
}

type NetworkCompatibility struct {
	Network          string `json:"network"`
	IsCompatible     bool   `json:"isCompatible"`
	RetrievedVersion string `json:"retrievedVersion"`
	Error            error  `json:"error"`
}

func CheckSoftwareCompatibility(netStore NetworkStore, requestVersionFn RequestVersionFn) (*CheckSoftwareCompatibilityResponse, error) {
	networks, err := netStore.ListNetworks()
	// If there's an error we don't fail the command as the compatibility matrix
	// is just a nice to have.
	if err != nil {
		// Best-effort, so we don't fail.
		return nil, ErrCouldNotListNetworks
	}

	networksCompatibility := make([]NetworkCompatibility, 0, len(networks))

	coreVersion := coreversion.Get()
	coreVersionForComparison := onlyMajorAndMinor(coreVersion)

	for _, net := range networks {
		networkCompatibility := NetworkCompatibility{
			Network: net,
		}

		netConfig, err := netStore.GetNetwork(net)
		if err != nil {
			// Best-effort, so we don't fail.
			networkCompatibility.Error = err
			networksCompatibility = append(networksCompatibility, networkCompatibility)
			continue
		}

		if err := netConfig.EnsureCanConnectGRPCNode(); err != nil {
			// Best-effort, so we don't fail.
			networkCompatibility.Error = err
			networksCompatibility = append(networksCompatibility, networkCompatibility)
			continue
		}

		networkVersion, err := requestVersionFn(netConfig.API.GRPC.Hosts)
		if err != nil {
			// Best-effort, so we don't fail.
			networkCompatibility.Error = err
			networksCompatibility = append(networksCompatibility, networkCompatibility)
			continue
		}

		networkCompatibility.RetrievedVersion = networkVersion
		networkVersionForComparison := onlyMajorAndMinor(networkVersion)

		if networkVersionForComparison != coreVersionForComparison {
			networkCompatibility.IsCompatible = false
			networksCompatibility = append(networksCompatibility, networkCompatibility)
			continue
		}

		networkCompatibility.IsCompatible = true
		networksCompatibility = append(networksCompatibility, networkCompatibility)
	}

	// Ensure the output is determinist.
	sort.Slice(networksCompatibility, func(a, b int) bool {
		return networksCompatibility[a].Network < networksCompatibility[b].Network
	})

	return &CheckSoftwareCompatibilityResponse{
		NetworksCompatibility: networksCompatibility,
	}, nil
}

func onlyMajorAndMinor(version string) string {
	segments := strings.Split(version, ".")
	if len(segments) < 2 {
		// It doesn't seem to be a valid semantic version.
		return version
	}

	return segments[0] + "." + segments[1]
}
