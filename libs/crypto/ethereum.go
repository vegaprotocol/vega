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
	"github.com/ethereum/go-ethereum/common"
)

// EthereumChecksumAddress is a simple utility function
// to ensure all ethereum addresses used in vega are checksumed
// this expects a hex encoded string.
func EthereumChecksumAddress(s string) string {
	// as per docs the Hex method return EIP-55 compliant hex strings
	return common.HexToAddress(s).Hex()
}

// EthereumIsValidAddress returns whether the given string is a valid ethereum address.
func EthereumIsValidAddress(s string) bool {
	return common.IsHexAddress(s)
}
