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
	"fmt"
	"time"

	vgreflect "code.vegaprotocol.io/vega/libs/reflect"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type EVMChainConfig struct {
	chainID          string
	networkID        string
	confirmations    uint64
	collateralBridge EthereumContract
	multiSigControl  EthereumContract
	blockTime        time.Duration
}

func EVMChainConfigFromUntypedProto(v interface{}) (*EVMChainConfig, error) {
	cfg, err := toEVMChainConfigProto(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to EVMChainConfig proto: %w", err)
	}

	ethConfig, err := SecondaryConfigFromProto(cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't build EVMChainConfig: %w", err)
	}

	return ethConfig, nil
}

func SecondaryConfigFromProto(cfgProto *proto.EVMChainConfig) (*EVMChainConfig, error) {
	if err := CheckEVMChainConfig(cfgProto); err != nil {
		return nil, fmt.Errorf("invalid EVM chain configuration: %w", err)
	}

	cfg := &EVMChainConfig{
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

	if len(cfgProto.BlockTime) != 0 {
		bl, err := time.ParseDuration(cfgProto.BlockTime)
		if err != nil {
			return nil, fmt.Errorf("invalid EVM chain configuration, block_length: %w", err)
		}
		cfg.blockTime = bl
	}

	return cfg, nil
}

func (c *EVMChainConfig) BlockTime() time.Duration {
	return c.blockTime
}

func (c *EVMChainConfig) ChainID() string {
	return c.chainID
}

func (c *EVMChainConfig) NetworkID() string {
	return c.networkID
}

func (c *EVMChainConfig) Confirmations() uint64 {
	return c.confirmations
}

func (c *EVMChainConfig) CollateralBridge() EthereumContract {
	return c.collateralBridge
}

func (c *EVMChainConfig) MultiSigControl() EthereumContract {
	return c.multiSigControl
}

// CheckUntypedEVMChainConfig verifies the `v` parameter is a proto.EVMChainConfig
// struct and check if it's valid.
func CheckUntypedEVMChainConfig(v interface{}, _ interface{}) error {
	cfg, err := toEVMChainConfigProto(v)
	if err != nil {
		return err
	}

	return CheckEVMChainConfig(cfg)
}

// CheckEVMChainConfig verifies the proto.EVMChainConfig is valid.
func CheckEVMChainConfig(cfgProto *proto.EVMChainConfig) error {
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

	if len(cfgProto.BlockTime) != 0 {
		_, err := time.ParseDuration(cfgProto.BlockTime)
		if err != nil {
			return ErrInvalidBlockLengthDuration
		}
	}

	return nil
}

func toEVMChainConfigProto(v interface{}) (*proto.EVMChainConfig, error) {
	cfg, ok := v.(*proto.EVMChainConfig)
	if !ok {
		return nil, fmt.Errorf("type %q is not a EVMChainConfig proto", vgreflect.TypeName(v))
	}
	return cfg, nil
}
