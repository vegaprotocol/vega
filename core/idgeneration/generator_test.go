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

package idgeneration_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/idgeneration"

	"github.com/stretchr/testify/assert"
)

func TestGeneratorCreationFailsWithInvalidRootId(t *testing.T) {
}

func TestOrderIdGeneration(t *testing.T) {
	detID := "e1152cf235f6200ed0eb4598706821031d57403462c31a80b3cdd6b209bff2e6"
	gen := idgeneration.New(detID)

	assert.Equal(t, detID, gen.NextID())
	assert.NotEqual(t, detID, gen.NextID())
}
