package network_test

import (
	"testing"

	"code.vegaprotocol.io/vega/wallet/network"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Run("Ensure network can connect to a gRPC node fails", testEnsureNetworkCanConnectGRPCNodeFails)
}

func testEnsureNetworkCanConnectGRPCNodeFails(t *testing.T) {
	// given
	net := &network.Network{
		API: network.APIConfig{GRPC: network.HostConfig{
			Hosts: nil,
		}},
	}

	// when
	err := net.EnsureCanConnectGRPCNode()

	// then
	require.ErrorIs(t, err, network.ErrNetworkDoesNotHaveGRPCHostConfigured)
}
