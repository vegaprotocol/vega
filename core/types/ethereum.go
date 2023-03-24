// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
)

type EthereumConfig struct {
	chainID          string
	networkID        string
	collateralBridge EthereumContract
	multiSigControl  EthereumContract
	stakingBridge    EthereumContract
	vestingBridge    EthereumContract
	confirmations    uint64
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
