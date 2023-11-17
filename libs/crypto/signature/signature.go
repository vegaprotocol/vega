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

func RecoverEthereumAddress(message, signature []byte) (common.Address, error) {
	hash := ecrypto.Keccak256(message)

	if len(signature) <= ecrypto.RecoveryIDOffset {
		return common.Address{}, ErrEthInvalidSignature
	}

	// see reference in multisig control signature verification for more details
	if signature[ecrypto.RecoveryIDOffset] == 27 || signature[ecrypto.RecoveryIDOffset] == 28 {
		signature[ecrypto.RecoveryIDOffset] -= 27
	}

	// get the pubkey from the signature
	pubkey, err := ecrypto.SigToPub(hash, signature)
	if err != nil {
		return common.Address{}, err
	}

	// verify the signature
	signatureNoID := signature[:len(signature)-1]
	if !ecrypto.VerifySignature(ecrypto.CompressPubkey(pubkey), hash, signatureNoID) {
		return common.Address{}, ErrEthInvalidSignature
	}

	return ecrypto.PubkeyToAddress(*pubkey), nil
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
