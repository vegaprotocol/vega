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
	"crypto"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
)

type ed25519Sig struct{}

func newEd25519() *ed25519Sig {
	return &ed25519Sig{}
}

func (e *ed25519Sig) Sign(priv crypto.PrivateKey, buf []byte) ([]byte, error) {
	privBytes, ok := priv.([]byte)
	if !ok {
		return nil, ErrCouldNotCastPrivateKeyToBytes
	}
	// Avoid panic by checking key length
	if len(privBytes) != ed25519.PrivateKeySize {
		return nil, ErrBadED25519PrivateKeyLength
	}
	return ed25519.Sign(privBytes, vgcrypto.Hash(buf)), nil
}

func (e *ed25519Sig) Verify(pub crypto.PublicKey, message, sig []byte) (bool, error) {
	pubBytes, ok := pub.([]byte)
	if !ok {
		return false, ErrCouldNotCastPublicKeyToBytes
	}
	// Avoid panic by checking key length
	if len(pubBytes) != ed25519.PublicKeySize {
		return false, ErrBadED25519PublicKeyLength
	}
	return ed25519.Verify(pubBytes, vgcrypto.Hash(message), sig), nil
}

func (e *ed25519Sig) Name() string {
	return "vega/ed25519"
}

func (e *ed25519Sig) Version() uint32 {
	return 1
}
