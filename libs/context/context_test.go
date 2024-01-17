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

package context_test

import (
	"context"
	"strings"
	"testing"

	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/version"

	"github.com/stretchr/testify/assert"
)

func TestRestoreDataInContext(t *testing.T) {
	ctx := vgcontext.WithSnapshotInfo(context.Background(), "v0.74.0", true)
	assert.True(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0"))
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.1"))
	assert.True(t, vgcontext.InProgressSnapshotRestore(ctx))

	ctx = vgcontext.WithSnapshotInfo(context.Background(), "v0.74.0", false)
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0"))
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.1"))
	assert.True(t, vgcontext.InProgressSnapshotRestore(ctx))

	ctx = context.Background()
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0"))
	assert.False(t, vgcontext.InProgressUpgradeFrom(ctx, "v0.74.1"))
	assert.False(t, vgcontext.InProgressSnapshotRestore(ctx))
}

func TestUpgradeTo(t *testing.T) {
	current := strings.Split(strings.Split(version.Get(), "-")[0], "+")[0]
	data := []struct {
		from, to string
	}{
		{
			from: "v0.1.0",
			to:   "v0.1.1",
		},
		{
			from: current,
			to:   current + "2",
		},
		{
			from: current + "2",
			to:   current + "3",
		},
	}
	for _, d := range data {
		ctx := vgcontext.WithSnapshotInfo(context.Background(), d.from, true)
		assert.True(t, vgcontext.InProgressUpgradeFrom(ctx, d.from))
		assert.True(t, vgcontext.InProgressUpgradeTo(ctx, current))
		assert.False(t, vgcontext.InProgressUpgradeTo(ctx, d.to))
	}
	// for completeness:
	assert.False(t, vgcontext.InProgressUpgradeTo(context.Background(), current))
	assert.False(t, vgcontext.InProgressUpgradeFrom(context.Background(), current))
}
