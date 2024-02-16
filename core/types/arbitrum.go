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

type ArbitrumConfig struct {
	chainID          string
	networkID        string
	confirmations    uint64
	collateralBridge EthereumContract
	multiSigControl  EthereumContract
}

func ArbitrumConfigFromUntypedProto(v interface{}) (*ArbitrumConfig, error) {
	cfg, err := toArbitrumConfigProto(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to ArbitrumConfig proto: %w", err)
	}

	ethConfig, err := ArbitrumConfigFromProto(cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't build ArbitrumConfig: %w", err)
	}

	return ethConfig, nil
}

func ArbitrumConfigFromProto(cfgProto *proto.ArbitrumConfig) (*ArbitrumConfig, error) {
	if err := CheckArbitrumConfig(cfgProto); err != nil {
		return nil, fmt.Errorf("invalid Arbitrum configuration: %w", err)
	}

	cfg := &ArbitrumConfig{
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

func (c *ArbitrumConfig) ChainID() string {
	return c.chainID
}

func (c *ArbitrumConfig) NetworkID() string {
	return c.networkID
}

func (c *ArbitrumConfig) Confirmations() uint64 {
	return c.confirmations
}

func (c *ArbitrumConfig) CollateralBridge() EthereumContract {
	return c.collateralBridge
}

func (c *ArbitrumConfig) MultiSigControl() EthereumContract {
	return c.multiSigControl
}

// CheckUntypedArbitrumConfig verifies the `v` parameter is a proto.ArbitrumConfig
// struct and check if it's valid.
func CheckUntypedArbitrumConfig(v interface{}, _ interface{}) error {
	cfg, err := toArbitrumConfigProto(v)
	if err != nil {
		return err
	}

	return CheckArbitrumConfig(cfg)
}

// CheckArbitrumConfig verifies the proto.ArbitrumConfig is valid.
func CheckArbitrumConfig(cfgProto *proto.ArbitrumConfig) error {
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

func toArbitrumConfigProto(v interface{}) (*proto.ArbitrumConfig, error) {
	cfg, ok := v.(*proto.ArbitrumConfig)
	if !ok {
		return nil, fmt.Errorf("type %q is not a ArbitrumConfig proto", vgreflect.TypeName(v))
	}
	return cfg, nil
}
