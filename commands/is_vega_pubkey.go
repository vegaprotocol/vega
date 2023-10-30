// Copyright (C) 2023  Gobalsky Labs Limited
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

package commands

import (
	"encoding/hex"
	"errors"
)

const (
	vegaPublicKeyLen = 64
	vegaIDLen        = 64
)

var (
	ErrShouldBeAValidVegaPublicKey = errors.New("should be a valid vega public key")
	ErrShouldBeAValidVegaID        = errors.New("should be a valid Vega ID")
)

// IsVegaPublicKey check if a string is a valid Vega public key.
// A public key is a string of 64 characters containing only hexadecimal characters.
// Despite being similar to the function IsVegaID, the Vega ID and public
// key are different concept that generated, and used differently.
func IsVegaPublicKey(key string) bool {
	_, err := hex.DecodeString(key)
	return len(key) == vegaPublicKeyLen && err == nil
}

// IsVegaID check if a string is a valid Vega public key.
// An ID is a string of 64 characters containing only hexadecimal characters.
// Despite being similar to the function IsVegaPublicKey, the Vega ID and public
// key are different concept that generated, and used differently.
func IsVegaID(id string) bool {
	_, err := hex.DecodeString(id)
	return len(id) == vegaIDLen && err == nil
}
