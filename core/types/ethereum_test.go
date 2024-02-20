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

package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"

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
	t.Run("Basic checks on fields of the EVM config", testEVMConfigBasic)
	t.Run("Check that a network cannot appear twice in the config", testEVMConfigRejectDuplicateFields)
	t.Run("Check that an EVM config can not be removed", testEVMAmendOrAppendOnly)
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

func testEVMConfigBasic(t *testing.T) {
	tcs := []struct {
		name          string
		chainID       string
		networkID     string
		confirmations uint32
		blockInterval uint64
		expect        error
	}{
		{
			name:          "hello",
			chainID:       "11",
			networkID:     "12",
			confirmations: 1,
			blockInterval: 1,
		},
		{
			chainID:       "11",
			networkID:     "12",
			confirmations: 1,
			blockInterval: 1,
			expect:        types.ErrMissingNetworkName,
		},
		{
			name:          "hello",
			networkID:     "12",
			confirmations: 1,
			blockInterval: 1,
			expect:        types.ErrMissingChainID,
		},
		{
			name:          "hello",
			chainID:       "11",
			confirmations: 1,
			blockInterval: 1,
			expect:        types.ErrMissingNetworkID,
		},
		{
			name:          "hello",
			chainID:       "11",
			networkID:     "12",
			confirmations: 0,
			blockInterval: 1,
			expect:        types.ErrConfirmationsMustBeHigherThan0,
		},
		{
			name:          "hello",
			chainID:       "11",
			networkID:     "12",
			confirmations: 1,
			blockInterval: 0,
			expect:        types.ErrBlockIntervalMustBeHigherThan0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			cfgs := &proto.EthereumL2Configs{
				Configs: []*proto.EthereumL2Config{
					{
						NetworkId:     tc.networkID,
						ChainId:       tc.chainID,
						Name:          tc.name,
						Confirmations: tc.confirmations,
						BlockInterval: tc.blockInterval,
					},
				},
			}

			// when
			err := types.CheckEthereumL2Configs(cfgs, nil)

			// then
			require.ErrorIs(t, tc.expect, err)
		})
	}
}

func testEVMConfigRejectDuplicateFields(t *testing.T) {
	cfgs := validEVMConfigs()

	original := proto.EthereumL2Config{
		NetworkId:     "999",
		ChainId:       "9999",
		Name:          "999999",
		Confirmations: 100,
		BlockInterval: 1,
	}

	c := original
	cfgs.Configs = append(cfgs.Configs, &c)
	err := types.CheckEthereumL2Configs(cfgs, nil)
	require.ErrorIs(t, nil, err)

	c.Name = cfgs.Configs[0].Name
	err = types.CheckEthereumL2Configs(cfgs, nil)
	require.ErrorIs(t, types.ErrDuplicateNetworkName, err)
	c.Name = original.Name

	c.ChainId = cfgs.Configs[0].ChainId
	err = types.CheckEthereumL2Configs(cfgs, nil)
	require.ErrorIs(t, types.ErrDuplicateChainID, err)
	c.ChainId = original.ChainId

	c.NetworkId = cfgs.Configs[0].NetworkId
	err = types.CheckEthereumL2Configs(cfgs, nil)
	require.ErrorIs(t, types.ErrDuplicateNetworkID, err)
}

func testEVMAmendOrAppendOnly(t *testing.T) {
	cfgs1 := validEVMConfigs()
	cfgs2 := validEVMConfigs()

	// update to itself
	err := types.CheckEthereumL2Configs(cfgs1, cfgs2)
	require.ErrorIs(t, nil, err)

	// change only confirmations
	cfgs2.Configs[0].Confirmations += 1
	err = types.CheckEthereumL2Configs(cfgs1, cfgs2)
	require.ErrorIs(t, nil, err)

	// try to change the name
	cfgs2 = validEVMConfigs()
	cfgs2.Configs[0].Name += "hello"
	err = types.CheckEthereumL2Configs(cfgs1, cfgs2)
	require.ErrorIs(t, types.ErrCanOnlyAmendedConfirmationsAndBlockInterval, err)

	// try to change the chainID
	cfgs2 = validEVMConfigs()
	cfgs2.Configs[0].ChainId += "1"
	err = types.CheckEthereumL2Configs(cfgs1, cfgs2)
	require.ErrorIs(t, types.ErrCanOnlyAmendedConfirmationsAndBlockInterval, err)

	// try to change the networkID (counts as a remove)
	cfgs2 = validEVMConfigs()
	cfgs2.Configs[0].NetworkId += "1"
	err = types.CheckEthereumL2Configs(cfgs1, cfgs2)
	require.ErrorIs(t, types.ErrCannotRemoveL2Config, err)

	// add a new config that clashes with existing
	cfgs2 = validEVMConfigs()
	new_cfg := proto.EthereumL2Config{
		NetworkId:     cfgs2.Configs[0].NetworkId,
		ChainId:       "9999",
		Name:          "999999",
		Confirmations: 100,
	}
	cfgs2.Configs = append(cfgs2.Configs, &new_cfg)
	err = types.CheckEthereumL2Configs(cfgs1, cfgs2)
	require.ErrorIs(t, types.ErrCanOnlyAmendedConfirmationsAndBlockInterval, err)
}

func validEthereumConfig() *proto.EthereumConfig {
	return &proto.EthereumConfig{
		NetworkId: "1",
		ChainId:   "1",
		CollateralBridgeContract: &proto.EthereumContractConfig{
			Address: "0x1234",
		},
		MultisigControlContract: &proto.EthereumContractConfig{
			Address:               "0x1234",
			DeploymentBlockHeight: 789,
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

func validEVMConfigs() *proto.EthereumL2Configs {
	return &proto.EthereumL2Configs{
		Configs: []*proto.EthereumL2Config{
			{
				NetworkId:     "1",
				ChainId:       "2",
				Name:          "hello",
				Confirmations: 12,
				BlockInterval: 1,
			},
			{
				NetworkId:     "2",
				ChainId:       "3",
				Name:          "helloagain",
				Confirmations: 13,
				BlockInterval: 1,
			},
		},
	}
}
