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
	"encoding/hex"
	"fmt"

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
	hash := msg

	var sig []byte
	var err error

	if signer.Algo() == "clef" {
		sig, err = signer.Sign(hash)
	} else {
		// hash our message before signing it
		hash = crypto.Keccak256(msg)
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
