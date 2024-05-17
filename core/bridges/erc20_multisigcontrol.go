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
	"encoding/hex"
	"fmt"

	crypto "code.vegaprotocol.io/vega/libs/crypto/signature"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type ERC20MultiSigControl struct {
	signer  Signer
	chainID string
	v1      bool
}

func NewERC20MultiSigControl(signer Signer, chainID string, v1 bool) *ERC20MultiSigControl {
	return &ERC20MultiSigControl{
		signer:  signer,
		chainID: chainID,
		v1:      v1,
	}
}

func (e *ERC20MultiSigControl) BurnNonce(
	submitter string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
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
			Name: "nonce",
			Type: typU256,
		},
		{
			Name: "func_name",
			Type: typString,
		},
	})

	buf, err := args.Pack([]interface{}{nonce.BigInt(), "burn_nonce"}...)
	if err != nil {
		return nil, err
	}

	msg, err := packScheme(buf, submitter, e.chainID, e.v1)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e *ERC20MultiSigControl) SetThreshold(
	newThreshold uint16,
	submitter string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	typString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typU16, err := abi.NewType("uint16", "", nil)
	if err != nil {
		return nil, err
	}
	typU256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments([]abi.Argument{
		{
			Name: "new_threshold",
			Type: typU16,
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

	buf, err := args.Pack([]interface{}{newThreshold, nonce.BigInt(), "set_threshold"}...)
	if err != nil {
		return nil, err
	}

	msg, err := packScheme(buf, submitter, e.chainID, e.v1)
	if err != nil {
		return nil, err
	}
	return sign(e.signer, msg)
}

func (e *ERC20MultiSigControl) AddSigner(
	newSigner, submitter string,
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
			Name: "new_signer",
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

	newSignerAddr := ethcmn.HexToAddress(newSigner)
	buf, err := args.Pack([]interface{}{newSignerAddr, nonce.BigInt(), "add_signer"}...)
	if err != nil {
		return nil, err
	}

	msg, err := packScheme(buf, submitter, e.chainID, e.v1)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e *ERC20MultiSigControl) VerifyRemoveSigner(
	oldSigner, submitter string,
	nonce *num.Uint,
	signatures string,
) ([]string, error) {
	msg, err := e.buildRemoveSignerMessage(
		oldSigner, submitter, nonce,
	)
	if err != nil {
		return nil, err
	}

	addresses := []string{}
	var hexCurrent string
	signatures = signatures[2:]
	for len(signatures) > 0 {
		hexCurrent, signatures = signatures[0:130], signatures[130:]
		current, err := hex.DecodeString(hexCurrent)
		if err != nil {
			return nil, fmt.Errorf("invalid signature format: %w", err)
		}

		address, err := crypto.RecoverEthereumAddress(msg, current)
		if err != nil {
			return nil, fmt.Errorf("error recovering ethereum address: %w", err)
		}

		addresses = append(addresses, address.Hex())
	}

	return addresses, nil
}

func (e *ERC20MultiSigControl) RemoveSigner(
	oldSigner, submitter string,
	nonce *num.Uint,
) (*SignaturePayload, error) {
	msg, err := e.buildRemoveSignerMessage(
		oldSigner, submitter, nonce,
	)
	if err != nil {
		return nil, err
	}

	return sign(e.signer, msg)
}

func (e *ERC20MultiSigControl) buildRemoveSignerMessage(
	oldSigner, submitter string,
	nonce *num.Uint,
) ([]byte, error) {
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
			Name: "old_signer",
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

	oldSignerAddr := ethcmn.HexToAddress(oldSigner)
	buf, err := args.Pack([]interface{}{oldSignerAddr, nonce.BigInt(), "remove_signer"}...)
	if err != nil {
		return nil, err
	}

	return packScheme(buf, submitter, e.chainID, e.v1)
}
