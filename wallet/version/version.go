package version

import (
	"sort"

	vgversion "code.vegaprotocol.io/vega/libs/version"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/network"
)

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

type GetVersionResponse struct {
	Version               string                 `json:"version"`
	GitHash               string                 `json:"gitHash"`
	NetworksCompatibility []NetworkCompatibility `json:"networksCompatibility"`
}

type NetworkCompatibility struct {
	Network          string `json:"network"`
	IsCompatible     bool   `json:"isCompatible"`
	RetrievedVersion string `json:"retrievedVersion"`
	Error            error  `json:"error"`
}

func GetVersionInfo(netStore NetworkStore, requestVersionFn RequestVersionFn) *GetVersionResponse {
	response := &GetVersionResponse{
		Version: coreversion.Get(),
		GitHash: coreversion.GetCommitHash(),
	}

	networks, err := netStore.ListNetworks()
	// If there's an error we don't fail the command as the compatibility matrix
	// is just a nice to have.
	if err != nil {
		// Best-effort, so we don't fail.
		return response
	}

	networksCompatibility := make([]NetworkCompatibility, 0, len(networks))

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

		if networkVersion != coreversion.Get() {
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

	response.NetworksCompatibility = networksCompatibility

	return response
}
