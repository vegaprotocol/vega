package version

import (
	"fmt"

	vgversion "code.vegaprotocol.io/vega/libs/version"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/network"
)

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
	Version               string            `json:"version"`
	GitHash               string            `json:"gitHash"`
	NetworksCompatibility map[string]string `json:"networksCompatibility"`
}

func GetVersionInfo(netStore NetworkStore, requestVersionFn RequestVersionFn) (*GetVersionResponse, error) {
	networksCompatibility := map[string]string{}
	networks, err := netStore.ListNetworks()
	// If there's an error we don't fail the command as the compatibility matrix
	// is just a nice to have.
	if err == nil {
		for _, net := range networks {
			// Default value.
			networksCompatibility[net] = "unable to determine"

			netConfig, err := netStore.GetNetwork(net)
			if err != nil {
				// We are in a best-effort, so we don't fail.
				continue
			}

			if err := netConfig.EnsureCanConnectGRPCNode(); err != nil {
				// We are in a best-effort, so we don't fail.
				continue
			}

			networkVersion, err := requestVersionFn(netConfig.API.GRPC.Hosts)
			if err != nil {
				// We are in a best-effort, so we don't fail.
				continue
			}

			if networkVersion != coreversion.Get() {
				networksCompatibility[net] = fmt.Sprintf("not compatible: network is running version %s but wallet software has version %s", networkVersion, coreversion.Get())
				continue
			}
			networksCompatibility[net] = "compatible"
		}
	}

	return &GetVersionResponse{
		Version:               coreversion.Get(),
		GitHash:               coreversion.GetCommitHash(),
		NetworksCompatibility: networksCompatibility,
	}, nil
}
