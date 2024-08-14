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

package bridges

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type ERC20AssetPool struct {
	signer   Signer
	poolAddr string
	chainID  string
	v1       bool
}

func NewERC20AssetPool(signer Signer, poolAddr, chainID string, v1 bool) *ERC20AssetPool {
	return &ERC20AssetPool{
		signer:   signer,
		poolAddr: poolAddr,
		chainID:  chainID,
		v1:       v1,
	}
}

func (e ERC20AssetPool) SetBridgeAddress(
	newAddress string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "address",
			Type: typAddr,
		},
		{
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	newAddressEth := ethcmn.HexToAddress(newAddress)
	buf, err := args.Pack([]interface{}{
		newAddressEth, nonce.BigInt(), "setBridgeAddress",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.poolAddr, e.chainID, e.v1)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}

func (e ERC20AssetPool) SetMultiSigControl(
	newAddress string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "address",
			Type: typAddr,
		},
		{
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	newAddressEth := ethcmn.HexToAddress(newAddress)
	buf, err := args.Pack([]interface{}{
		newAddressEth, nonce.BigInt(), "setMultisigControl",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packScheme(buf, e.poolAddr, e.chainID, e.v1)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}
