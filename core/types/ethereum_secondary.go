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

	vgreflect "code.vegaprotocol.io/vega/libs/reflect"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type SecondaryEthereumConfig struct {
	chainID          string
	networkID        string
	confirmations    uint64
	collateralBridge EthereumContract
	multiSigControl  EthereumContract
}

func SecondaryEthereumConfigFromUntypedProto(v interface{}) (*SecondaryEthereumConfig, error) {
	cfg, err := toSecondaryEthereumConfigProto(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to SecondaryEthereumConfig proto: %w", err)
	}

	ethConfig, err := SecondaryConfigFromProto(cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't build SecondaryEthereumConfig: %w", err)
	}

	return ethConfig, nil
}

func SecondaryConfigFromProto(cfgProto *proto.SecondaryEthereumConfig) (*SecondaryEthereumConfig, error) {
	if err := CheckSecondaryEthereumConfig(cfgProto); err != nil {
		return nil, fmt.Errorf("invalid second ethereum configuration: %w", err)
	}

	cfg := &SecondaryEthereumConfig{
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

	return cfg, nil
}

func (c *SecondaryEthereumConfig) ChainID() string {
	return c.chainID
}

func (c *SecondaryEthereumConfig) NetworkID() string {
	return c.networkID
}

func (c *SecondaryEthereumConfig) Confirmations() uint64 {
	return c.confirmations
}

func (c *SecondaryEthereumConfig) CollateralBridge() EthereumContract {
	return c.collateralBridge
}

func (c *SecondaryEthereumConfig) MultiSigControl() EthereumContract {
	return c.multiSigControl
}

// CheckUntypedSecondaryEthereumConfig verifies the `v` parameter is a proto.SecondaryEthereumConfig
// struct and check if it's valid.
func CheckUntypedSecondaryEthereumConfig(v interface{}, _ interface{}) error {
	cfg, err := toSecondaryEthereumConfigProto(v)
	if err != nil {
		return err
	}

	return CheckSecondaryEthereumConfig(cfg)
}

// CheckSecondaryEthereumConfig verifies the proto.SecondaryEthereumConfig is valid.
func CheckSecondaryEthereumConfig(cfgProto *proto.SecondaryEthereumConfig) error {
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

	return nil
}

func toSecondaryEthereumConfigProto(v interface{}) (*proto.SecondaryEthereumConfig, error) {
	cfg, ok := v.(*proto.SecondaryEthereumConfig)
	if !ok {
		return nil, fmt.Errorf("type %q is not a SecondaryEthereumConfig proto", vgreflect.TypeName(v))
	}
	return cfg, nil
}
