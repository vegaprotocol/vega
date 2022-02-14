package types_test

import (
	"testing"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/require"
)

func TestEthereumConfig(t *testing.T) {
	t.Run("Valid EthereumConfig config succeeds", testValidEthereumConfigSucceeds)
	t.Run("Missing network ID fails", testMissingNetworkIDFails)
	t.Run("Missing chain ID fails", testMissingChainIDFails)
	t.Run("Mis-configured collateral bridge contract fails", testMisconfiguredCollateralBridgeFails)
	t.Run("Missing both staking and vesting contract addresses fails", testMissingBothStakingAndVestingContractAddressesFails)
	t.Run("At least one of staking of vesting contract addresses succeeds", testAtLeastOneOfStackingOrVestingContractAddressesSucceeds)
	t.Run("Confirmations set to 0 fails", testConfirmationsSetTo0Fails)
}

func testValidEthereumConfigSucceeds(t *testing.T) {
	// given
	cfg := validEthereumConfig()

	// when
	err := types.CheckEthereumConfig(cfg)

	// then
	require.NoError(t, err)
}

func testMissingNetworkIDFails(t *testing.T) {
	// given
	cfg := validEthereumConfig()
	cfg.NetworkId = ""

	// when
	err := types.CheckEthereumConfig(cfg)

	// then
	require.ErrorIs(t, err, types.ErrMissingNetworkID)
}

func testMissingChainIDFails(t *testing.T) {
	// given
	cfg := validEthereumConfig()
	cfg.ChainId = ""

	// when
	err := types.CheckEthereumConfig(cfg)

	// then
	require.ErrorIs(t, err, types.ErrMissingChainID)
}

func testMisconfiguredCollateralBridgeFails(t *testing.T) {
	tcs := []struct {
		name     string
		contract *proto.EthereumContractConfig
		error    error
	}{
		{
			name:     "without stakingContract configuration",
			contract: nil,
			error:    types.ErrMissingCollateralBridgeAddress,
		}, {
			name:     "without stakingContract address",
			contract: &proto.EthereumContractConfig{},
			error:    types.ErrMissingCollateralBridgeAddress,
		}, {
			name: "without stakingContract deployment",
			contract: &proto.EthereumContractConfig{
				Address:               "0x1234",
				DeploymentBlockHeight: 1234,
			},
			error: types.ErrUnsupportedCollateralBridgeDeploymentBlockHeight,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			cfg := validEthereumConfig()
			cfg.CollateralBridgeContract = tc.contract

			// when
			err := types.CheckEthereumConfig(cfg)

			// then
			require.ErrorIs(t, err, tc.error)
		})
	}
}

func testMissingBothStakingAndVestingContractAddressesFails(t *testing.T) {
	tcs := []struct {
		name            string
		stakingContract *proto.EthereumContractConfig
		vestingContract *proto.EthereumContractConfig
		error           error
	}{
		{
			name:            "without staking nor vesting contract configuration",
			stakingContract: nil,
			vestingContract: nil,
			error:           types.ErrAtLeastOneOfStakingOrVestingBridgeAddressMustBeSet,
		}, {
			name:            "with unset staking address and no vesting contract configuration",
			stakingContract: &proto.EthereumContractConfig{},
			vestingContract: nil,
			error:           types.ErrAtLeastOneOfStakingOrVestingBridgeAddressMustBeSet,
		}, {
			name:            "with unset vesting address and no staking contract configuration",
			stakingContract: nil,
			vestingContract: &proto.EthereumContractConfig{},
			error:           types.ErrAtLeastOneOfStakingOrVestingBridgeAddressMustBeSet,
		}, {
			name:            "with both unset staking and vesting address",
			stakingContract: &proto.EthereumContractConfig{},
			vestingContract: &proto.EthereumContractConfig{},
			error:           types.ErrAtLeastOneOfStakingOrVestingBridgeAddressMustBeSet,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			cfg := validEthereumConfig()
			cfg.StakingBridgeContract = tc.stakingContract
			cfg.TokenVestingContract = tc.vestingContract

			// when
			err := types.CheckEthereumConfig(cfg)

			// then
			require.ErrorIs(t, err, tc.error)
		})
	}
}

func testAtLeastOneOfStackingOrVestingContractAddressesSucceeds(t *testing.T) {
	tcs := []struct {
		name            string
		stakingContract *proto.EthereumContractConfig
		vestingContract *proto.EthereumContractConfig
	}{
		{
			name: "with staking address but no vesting contract configuration",
			stakingContract: &proto.EthereumContractConfig{
				Address: "0x1234",
			},
			vestingContract: nil,
		}, {
			name:            "with vesting address but no staking contract configuration",
			stakingContract: nil,
			vestingContract: &proto.EthereumContractConfig{
				Address: "0x1234",
			},
		}, {
			name: "with both staking and vesting address",
			stakingContract: &proto.EthereumContractConfig{
				Address: "0x1234",
			},
			vestingContract: &proto.EthereumContractConfig{
				Address: "0x9876",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			cfg := validEthereumConfig()
			cfg.StakingBridgeContract = tc.stakingContract

			// when
			err := types.CheckEthereumConfig(cfg)

			// then
			require.NoError(t, err)
		})
	}
}

func testConfirmationsSetTo0Fails(t *testing.T) {
	// given
	cfg := validEthereumConfig()
	cfg.Confirmations = 0

	// when
	err := types.CheckEthereumConfig(cfg)

	// then
	require.ErrorIs(t, err, types.ErrConfirmationsMustBeHigherThan0)
}

func validEthereumConfig() *proto.EthereumConfig {
	return &proto.EthereumConfig{
		NetworkId: "1",
		ChainId:   "1",
		CollateralBridgeContract: &proto.EthereumContractConfig{
			Address: "0x1234",
		},
		Confirmations: 3,
		StakingBridgeContract: &proto.EthereumContractConfig{
			Address:               "0x1234",
			DeploymentBlockHeight: 987,
		},
		TokenVestingContract: &proto.EthereumContractConfig{
			Address:               "0x1234",
			DeploymentBlockHeight: 567,
		},
	}
}
