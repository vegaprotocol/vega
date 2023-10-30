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

package helpers

import (
	"math/rand"
	"time"
)

type Block struct {
	Duration int64
	Variance float64
}

func NewBlock() *Block {
	return &Block{
		Duration: 1,
	}
}

func (b Block) GetDuration() time.Duration {
	if b.Variance == 0 {
		return time.Duration(b.Duration) * time.Second
	}
	base := time.Duration(b.Duration) * time.Second
	factor := int64(b.Variance * float64(time.Second))
	// factor of 3, random 0-6 yields random number between -3 and +3
	offset := factor - rand.Int63n(2*factor)
	return base + time.Duration(offset)
}

func (b Block) GetStep(d time.Duration) time.Duration {
	if b.Variance == 0 {
		return d
	}
	factor := int64(b.Variance * float64(d))
	offset := factor - rand.Int63n(factor*2)
	return d + time.Duration(offset)
}
