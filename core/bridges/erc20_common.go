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

	"code.vegaprotocol.io/vega/core/nodewallets/eth/clef"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

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

func packBufAndSubmitter(
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
