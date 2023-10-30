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

package context_test

import (
	"context"
	"testing"

	vgcontext "code.vegaprotocol.io/vega/libs/context"

	"github.com/stretchr/testify/assert"
)

func TestRestoreDataInContext(t *testing.T) {
	ctx := vgcontext.WithSnapshotInfo(context.Background(), "v0.74.0", true)
	assert.True(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0"))
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.1"))

	ctx = vgcontext.WithSnapshotInfo(context.Background(), "v0.74.0", false)
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0"))
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.1"))

	ctx = context.Background()
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0"))
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.1"))
}
