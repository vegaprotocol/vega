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

package wallet

import (
	"crypto/ed25519"
	"encoding/hex"

	"code.vegaprotocol.io/vega/wallet/crypto"
)

type HDMasterKeyPair struct {
	publicKey  *key
	privateKey *key
	algo       crypto.SignatureAlgorithm
}

func NewHDMasterKeyPair(
	publicKey ed25519.PublicKey,
	privateKey ed25519.PrivateKey,
) (*HDMasterKeyPair, error) {
	algo, err := crypto.NewSignatureAlgorithm(crypto.Ed25519, 1)
	if err != nil {
		return nil, err
	}

	return &HDMasterKeyPair{
		publicKey: &key{
			bytes:   publicKey,
			encoded: hex.EncodeToString(publicKey),
		},
		privateKey: &key{
			bytes:   privateKey,
			encoded: hex.EncodeToString(privateKey),
		},
		algo: algo,
	}, nil
}

func (k *HDMasterKeyPair) PublicKey() string {
	return k.publicKey.encoded
}

func (k *HDMasterKeyPair) PrivateKey() string {
	return k.privateKey.encoded
}

func (k *HDMasterKeyPair) AlgorithmVersion() uint32 {
	return k.algo.Version()
}

func (k *HDMasterKeyPair) AlgorithmName() string {
	return k.algo.Name()
}

func (k *HDMasterKeyPair) SignAny(data []byte) ([]byte, error) {
	return k.algo.Sign(k.privateKey.bytes, data)
}

func (k *HDMasterKeyPair) Sign(data []byte) (*Signature, error) {
	sig, err := k.algo.Sign(k.privateKey.bytes, data)
	if err != nil {
		return nil, err
	}

	return &Signature{
		Value:   hex.EncodeToString(sig),
		Algo:    k.algo.Name(),
		Version: k.algo.Version(),
	}, nil
}
