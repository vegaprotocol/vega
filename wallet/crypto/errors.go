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
)

var (
	// ErrBadED25519PrivateKeyLength is returned if a private key with incorrect length is supplied.
	ErrBadED25519PrivateKeyLength = errors.New("bad ed25519 private key length")

	// ErrBadED25519PublicKeyLength is returned if a public key with incorrect length is supplied.
	ErrBadED25519PublicKeyLength = errors.New("bad ed25519 public key length")

	ErrCouldNotCastPrivateKeyToBytes = errors.New("couldn't cast private key to bytes")
	ErrCouldNotCastPublicKeyToBytes  = errors.New("couldn't cast public key to bytes")

	ErrSignatureIsNil = errors.New("signature is nil")
)
