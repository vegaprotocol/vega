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

package rand_test

import (
	"math/rand"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/stretchr/testify/assert"
)

func TestRandomHelpers(t *testing.T) {
	t.Run("Create a random string succeeds", testCreatingNewRandomStringSucceeds)
	t.Run("Create a random bytes succeeds", testCreatingNewRandomBytesSucceeds)
}

func testCreatingNewRandomStringSucceeds(t *testing.T) {
	size := rand.Intn(100)
	randomStr := vgrand.RandomStr(size)
	assert.Len(t, randomStr, size)
}

func testCreatingNewRandomBytesSucceeds(t *testing.T) {
	size := rand.Intn(100)
	randomBytes := vgrand.RandomBytes(size)
	assert.Len(t, randomBytes, size)
}
