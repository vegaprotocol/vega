// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package bridges

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type ERC20AssetPool struct {
	signer   Signer
	poolAddr string
}

func NewERC20AssetPool(signer Signer, poolAddr string) *ERC20AssetPool {
	return &ERC20AssetPool{
		signer:   signer,
		poolAddr: poolAddr,
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
		newAddressEth, nonce.BigInt(), "set_bridge_address",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.poolAddr)
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
		newAddressEth, nonce.BigInt(), "set_multisig_control",
	}...)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	msg, err := packBufAndSubmitter(buf, e.poolAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't pack abi message: %w", err)
	}

	return sign(e.signer, msg)
}
