// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package num_test

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntNumerics(t *testing.T) {
	n, err := num.NumericFromString("-10")
	require.NoError(t, err)
	assert.True(t, n.IsInt())
	assert.Equal(t, "-10", n.String())

	asInt := n.Int()
	require.NotNil(t, asInt)
	assert.Equal(t, n.String(), asInt.String())
}
