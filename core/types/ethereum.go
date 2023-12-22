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

package types

import (
	"errors"
	"fmt"

	vgreflect "code.vegaprotocol.io/vega/libs/reflect"
	proto "code.vegaprotocol.io/vega/protos/vega"

	ethcmn "github.com/ethereum/go-ethereum/common"
)

var (
	ErrMissingNetworkID                                   = errors.New("missing network ID in Ethereum config")
	ErrMissingChainID                                     = errors.New("missing chain ID in Ethereum config")
	ErrMissingCollateralBridgeAddress                     = errors.New("missing collateral bridge contract address in Ethereum config")
	ErrMissingMultiSigControlAddress                      = errors.New("missing multisig control contract address in Ethereum config")
	ErrUnsupportedCollateralBridgeDeploymentBlockHeight   = errors.New("setting collateral bridge contract deployment block height in Ethereum config is not supported")
	ErrAtLeastOneOfStakingOrVestingBridgeAddressMustBeSet = errors.New("at least one of the stacking bridge or token vesting contract addresses must be specified")
	ErrConfirmationsMustBeHigherThan0                     = errors.New("confirmation must be > 0 in Ethereum config")
	ErrMissingNetworkName                                 = errors.New("missing network name")
)

type EthereumConfig struct {
	chainID          string
	networkID        string
	confirmations    uint64
	collateralBridge EthereumContract
	multiSigControl  EthereumContract
	stakingBridge    EthereumContract
	vestingBridge    EthereumContract
}

func EthereumConfigFromUntypedProto(v interface{}) (*EthereumConfig, error) {
	cfg, err := toEthereumConfigProto(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to EthereumConfig proto: %w", err)
	}

	ethConfig, err := EthereumConfigFromProto(cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't build EthereumConfig: %w", err)
	}

	return ethConfig, nil
}

func EthereumConfigFromProto(cfgProto *proto.EthereumConfig) (*EthereumConfig, error) {
	if err := CheckEthereumConfig(cfgProto); err != nil {
		return nil, fmt.Errorf("invalid Ethereum configuration: %w", err)
	}

	cfg := &EthereumConfig{
		chainID:       cfgProto.ChainId,
		networkID:     cfgProto.NetworkId,
		confirmations: uint64(cfgProto.Confirmations),
		collateralBridge: EthereumContract{
			address: cfgProto.CollateralBridgeContract.Address,
		},
		multiSigControl: EthereumContract{
			address:               cfgProto.MultisigControlContract.Address,
			deploymentBlockHeight: cfgProto.MultisigControlContract.DeploymentBlockHeight,
		},
	}

	if cfgProto.StakingBridgeContract != nil {
		cfg.stakingBridge = EthereumContract{
			address:               cfgProto.StakingBridgeContract.Address,
			deploymentBlockHeight: cfgProto.StakingBridgeContract.DeploymentBlockHeight,
		}
	}

	if cfgProto.TokenVestingContract != nil {
		cfg.vestingBridge = EthereumContract{
			address:               cfgProto.TokenVestingContract.Address,
			deploymentBlockHeight: cfgProto.TokenVestingContract.DeploymentBlockHeight,
		}
	}

	return cfg, nil
}

func (c *EthereumConfig) ChainID() string {
	return c.chainID
}

func (c *EthereumConfig) NetworkID() string {
	return c.networkID
}

func (c *EthereumConfig) Confirmations() uint64 {
	return c.confirmations
}

func (c *EthereumConfig) CollateralBridge() EthereumContract {
	return c.collateralBridge
}

func (c *EthereumConfig) MultiSigControl() EthereumContract {
	return c.multiSigControl
}

func (c *EthereumConfig) StakingBridge() EthereumContract {
	return c.stakingBridge
}

func (c *EthereumConfig) VestingBridge() EthereumContract {
	return c.vestingBridge
}

// StakingBridgeAddresses returns the registered staking bridge addresses. It
// might return the staking bridge, or the token vesting, or both contract
// address. The vesting contract can also be used to get information needed by
// the staking engine.
func (c *EthereumConfig) StakingBridgeAddresses() []ethcmn.Address {
	var addresses []ethcmn.Address

	if c.stakingBridge.HasAddress() {
		addresses = append(addresses, c.stakingBridge.Address())
	}
	if c.vestingBridge.HasAddress() {
		addresses = append(addresses, c.vestingBridge.Address())
	}

	return addresses
}

type EthereumContract struct {
	address               string
	deploymentBlockHeight uint64
}

func (c EthereumContract) DeploymentBlockHeight() uint64 {
	return c.deploymentBlockHeight
}

func (c EthereumContract) HasAddress() bool {
	return len(c.address) > 0
}

func (c EthereumContract) Address() ethcmn.Address {
	return ethcmn.HexToAddress(c.address)
}

func (c EthereumContract) HexAddress() string {
	return c.address
}

// CheckUntypedEthereumConfig verifies the `v` parameter is a proto.EthereumConfig
// struct and check if it's valid.
func CheckUntypedEthereumConfig(v interface{}) error {
	cfg, err := toEthereumConfigProto(v)
	if err != nil {
		return err
	}

	return CheckEthereumConfig(cfg)
}

func CheckUntypedEthereumL2Configs(v interface{}) error {
	cfg, err := toEthereumL2ConfigsProto(v)
	if err != nil {
		return err
	}

	return CheckEthereumL2Configs(cfg)
}

// CheckEthereumConfig verifies the proto.EthereumConfig is valid.
func CheckEthereumConfig(cfgProto *proto.EthereumConfig) error {
	if len(cfgProto.NetworkId) == 0 {
		return ErrMissingNetworkID
	}

	if len(cfgProto.ChainId) == 0 {
		return ErrMissingChainID
	}

	if cfgProto.Confirmations == 0 {
		return ErrConfirmationsMustBeHigherThan0
	}

	noMultiSigControlSetUp := cfgProto.MultisigControlContract == nil || len(cfgProto.MultisigControlContract.Address) == 0
	if noMultiSigControlSetUp {
		return ErrMissingMultiSigControlAddress
	}

	noCollateralBridgeSetUp := cfgProto.CollateralBridgeContract == nil || len(cfgProto.CollateralBridgeContract.Address) == 0
	if noCollateralBridgeSetUp {
		return ErrMissingCollateralBridgeAddress
	}
	if cfgProto.CollateralBridgeContract.DeploymentBlockHeight != 0 {
		return ErrUnsupportedCollateralBridgeDeploymentBlockHeight
	}

	noStakingBridgeSetUp := cfgProto.StakingBridgeContract == nil || len(cfgProto.StakingBridgeContract.Address) == 0
	noVestingBridgeSetUp := cfgProto.TokenVestingContract == nil || len(cfgProto.TokenVestingContract.Address) == 0
	if noStakingBridgeSetUp && noVestingBridgeSetUp {
		return ErrAtLeastOneOfStakingOrVestingBridgeAddressMustBeSet
	}

	return nil
}

func toEthereumConfigProto(v interface{}) (*proto.EthereumConfig, error) {
	cfg, ok := v.(*proto.EthereumConfig)
	if !ok {
		return nil, fmt.Errorf("type \"%s\" is not a EthereumConfig proto", vgreflect.TypeName(v))
	}
	return cfg, nil
}

type EthereumL2Configs struct {
	Configs []EthereumL2Config
}

type EthereumL2Config struct {
	ChainID       string
	NetworkID     string
	Confirmations uint64
	Name          string
}

func toEthereumL2ConfigsProto(v interface{}) (*proto.EthereumL2Configs, error) {
	cfg, ok := v.(*proto.EthereumL2Configs)
	if !ok {
		return nil, fmt.Errorf("type \"%s\" is not a EthereumL2Configs proto", vgreflect.TypeName(v))
	}
	return cfg, nil
}

func EthereumL2ConfigsFromUntypedProto(v interface{}) (*EthereumL2Configs, error) {
	cfg, err := toEthereumL2ConfigsProto(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to EthereumL2Configs proto: %w", err)
	}

	ethConfig, err := EthereumL2ConfigsFromProto(cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't build EthereumL2Configs: %w", err)
	}

	return ethConfig, nil
}

func EthereumL2ConfigsFromProto(cfgProto *proto.EthereumL2Configs) (*EthereumL2Configs, error) {
	if err := CheckEthereumL2Configs(cfgProto); err != nil {
		return nil, fmt.Errorf("invalid Ethereum configuration: %w", err)
	}

	cfg := &EthereumL2Configs{}
	for _, v := range cfgProto.Configs {
		cfg.Configs = append(cfg.Configs, EthereumL2Config{
			NetworkID:     v.NetworkId,
			ChainID:       v.ChainId,
			Name:          v.Name,
			Confirmations: uint64(v.Confirmations),
		})
	}

	return cfg, nil
}

// CheckEthereumConfig verifies the proto.EthereumConfig is valid.
func CheckEthereumL2Configs(cfgProto *proto.EthereumL2Configs) error {
	for _, v := range cfgProto.Configs {
		if len(v.NetworkId) == 0 {
			return ErrMissingNetworkID
		}

		if len(v.ChainId) == 0 {
			return ErrMissingChainID
		}

		if v.Confirmations == 0 {
			return ErrConfirmationsMustBeHigherThan0
		}

		if len(v.Name) == 0 {
			return ErrMissingNetworkName
		}
	}

	return nil
}
