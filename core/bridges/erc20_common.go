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
	"bytes"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/core/nodewallets/eth/clef"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var prefix = []byte{0x19}

type Signer interface {
	Sign([]byte) ([]byte, error)
	Algo() string
}

type SignaturePayload struct {
	Message   Bytes
	Signature Bytes
}

type Bytes []byte

func (b Bytes) Bytes() []byte {
	return b
}

func (b Bytes) Hex() string {
	return hex.EncodeToString(b)
}

func packScheme(
	buf []byte, submitter, chainID string, v1 bool,
) ([]byte, error) {
	if v1 {
		return packSchemeV1(buf, submitter)
	}
	return packSchemeV2(buf, submitter, chainID)
}

// packSchemeV1 returns the payload to be hashed and signed where
// payload = abi.encode(message, msg.sender).
func packSchemeV1(
	buf []byte, submitter string,
) ([]byte, error) {
	typBytes, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}

	submitterAddr := ethcmn.HexToAddress(submitter)
	args2 := abi.Arguments([]abi.Argument{
		{
			Name: "bytes",
			Type: typBytes,
		},
		{
			Name: "address",
			Type: typAddr,
		},
	})

	return args2.Pack(buf, submitterAddr)
}

// packSchemeV2 returns the payload to be hashed and signed where
// payload = abi.encodePacked(bytes1(0x19), block.chainid, abi.encode(message, msg.sender))
// where abi.encodePacked is the concatenation of the individual byte slices.
func packSchemeV2(
	buf []byte, submitter, chainID string,
) ([]byte, error) {
	typBytes, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	typAddr, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}

	submitterAddr := ethcmn.HexToAddress(submitter)
	args := abi.Arguments([]abi.Argument{
		{
			Name: "bytes",
			Type: typBytes,
		},
		{
			Name: "address",
			Type: typAddr,
		},
	})

	// abi.encode(message, msg.sender)
	buf, err = args.Pack(buf, submitterAddr)
	if err != nil {
		return nil, err
	}

	// concat(prefix, chain-id, abi.encode(message, msg.sender))
	cid := num.MustUintFromString(chainID, 10).Bytes()
	return bytes.Join([][]byte{prefix, cid[:], buf}, nil), nil
}

func sign(signer Signer, msg []byte) (*SignaturePayload, error) {
	var sig []byte
	var err error

	if signer.Algo() == clef.ClefAlgoType {
		sig, err = signer.Sign(msg)
	} else {
		// hash our message before signing it
		hash := crypto.Keccak256(msg)
		sig, err = signer.Sign(hash)
	}

	if err != nil {
		return nil, fmt.Errorf("could not sign message with ethereum wallet: %w", err)
	}
	return &SignaturePayload{
		Message:   msg,
		Signature: sig,
	}, nil
}
