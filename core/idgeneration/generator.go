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

package idgeneration

import (
	"encoding/hex"

	"code.vegaprotocol.io/vega/libs/crypto"
)

// IDGenerator no mutex required, markets work deterministically, and sequentially.
type IDGenerator struct {
	nextIDBytes []byte
}

// New returns an idGenerator, and is used to abstract this type.
func New(rootID string) *IDGenerator { //revive:disable:unexported-return
	nextIDBytes, err := hex.DecodeString(rootID)
	if err != nil {
		panic("failed to create new deterministic id generator: " + err.Error())
	}

	return &IDGenerator{
		nextIDBytes: nextIDBytes,
	}
}

func (i *IDGenerator) NextID() string {
	if i == nil {
		panic("id generator instance is not initialised")
	}

	nextID := hex.EncodeToString(i.nextIDBytes)
	i.nextIDBytes = crypto.Hash(i.nextIDBytes)
	return nextID
}
