// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package idgeneration_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/idgeneration"
	"github.com/stretchr/testify/assert"
)

func TestGeneratorCreationFailsWithInvalidRootId(t *testing.T) {
}

func TestOrderIdGeneration(t *testing.T) {
	detId := "e1152cf235f6200ed0eb4598706821031d57403462c31a80b3cdd6b209bff2e6"
	gen := idgeneration.New(detId)

	assert.Equal(t, detId, gen.NextID())
	assert.NotEqual(t, detId, gen.NextID())
}
