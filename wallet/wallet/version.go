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

const (
	// Version1 identifies HD wallet with a key derivation version 1.
	Version1 = uint32(1)
	// Version2 identifies HD wallet with a key derivation version 2.
	Version2 = uint32(2)
	// LatestVersion is the latest version of Vega's HD wallet. Created wallets
	// are always pointing to the latest version.
	LatestVersion = Version2
)

// SupportedKeyDerivationVersions list of key derivation versions supported by
// Vega's HD wallet.
var SupportedKeyDerivationVersions = []uint32{Version1, Version2}

func IsKeyDerivationVersionSupported(v uint32) bool {
	return v == Version1 || v == Version2
}
