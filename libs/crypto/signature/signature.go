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

package crypto

import (
	"errors"
	"fmt"

	wcrypto "code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/ethereum/go-ethereum/common"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrAddressesDoesNotMatch = func(expected, recovered common.Address) error {
		return fmt.Errorf("addresses does not match, expected(%s) recovered(%s)", expected.Hex(), recovered.Hex())
	}
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrEthInvalidSignature = errors.New("invalid ethereum signature")
)

func VerifyEthereumSignature(message, signature []byte, hexAddress string) error {
	address := common.HexToAddress(hexAddress)
	hash := ecrypto.Keccak256(message)

	if len(signature) <= ecrypto.RecoveryIDOffset {
		return ErrEthInvalidSignature
	}

	// see reference in multisig control signature verification for more details
	if signature[ecrypto.RecoveryIDOffset] == 27 || signature[ecrypto.RecoveryIDOffset] == 28 {
		signature[ecrypto.RecoveryIDOffset] -= 27
	}

	// get the pubkey from the signature
	pubkey, err := ecrypto.SigToPub(hash, signature)
	if err != nil {
		return err
	}

	// verify the signature
	signatureNoID := signature[:len(signature)-1]
	if !ecrypto.VerifySignature(ecrypto.CompressPubkey(pubkey), hash, signatureNoID) {
		return ErrEthInvalidSignature
	}

	// ensure the signer is the expected ethereum wallet
	signerAddress := ecrypto.PubkeyToAddress(*pubkey)
	if address != signerAddress {
		return ErrAddressesDoesNotMatch(address, signerAddress)
	}

	return nil
}

func VerifyVegaSignature(message, signature, pubkey []byte) error {
	alg := wcrypto.NewEd25519()
	ok, err := alg.Verify(pubkey, message, signature)
	if err != nil {
		return err
	}

	if !ok {
		return ErrInvalidSignature
	}

	return nil
}
