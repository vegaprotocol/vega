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

package config

import "time"

const (
	TimeforwardAddress = "http://localhost:3101/api/v1/forwardtime"
	FaucetAddress      = "http://localhost:1790/api/v1/mint"
	GRCPAddress        = "localhost:3007"
	GoveranceAsset     = "VOTE"
	NormalAsset        = "XYZ"
	WalletFolder       = "nullchain-wallet"
	Passphrase         = "pin"
	BlockDuration      = time.Second
)
