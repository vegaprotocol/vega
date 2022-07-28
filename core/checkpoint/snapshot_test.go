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

package checkpoint_test

import (
	"context"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/core/libs/proto"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/stretchr/testify/require"
)

func TestCheckpointSnapshot(t *testing.T) {
	ctx := context.Background()

	e := getTestEngine(t)
	e.OnTimeElapsedUpdate(ctx, 10*time.Second)

	// This is 2022-02-04T11:50:12.655Z
	now := time.Unix(0, 1643975412655000000)

	// take a checkpoint so that we set the next-checkpoint time
	cp, err := e.Checkpoint(ctx, now)
	require.NoError(t, err)
	require.Nil(t, cp)

	// take a snapshot
	keys := e.Keys()
	data, _, err := e.GetState(keys[0])
	require.NoError(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data, snap)
	require.Nil(t, err)

	// Load the snapshot into a new engne
	snapEngine := getTestEngine(t)
	e.OnTimeElapsedUpdate(ctx, 10*time.Second) // netparam will get propagated into it
	_, err = snapEngine.LoadState(ctx, types.PayloadFromProto(snap))
	require.NoError(t, err)

	// this is 2022-02-04T11:50:22.591Z, if we failed to snapshot the microseconds we would be in a position where
	// restored-next-cp < now < original-next-cp
	now = time.Unix(0, 1643975422591000000)

	c1, err := e.Checkpoint(ctx, now)
	require.NoError(t, err)
	c2, err := snapEngine.Checkpoint(ctx, now)
	require.NoError(t, err)

	// Check that both engines do not take a checkpoint
	require.Nil(t, c1)
	require.Nil(t, c2)

	// shuffle forward
	now = now.Add(time.Second)
	c1, err = e.Checkpoint(ctx, now)
	require.NoError(t, err)
	c2, err = snapEngine.Checkpoint(ctx, now)
	require.NoError(t, err)

	// Check that they both now do
	require.NotNil(t, c1)
	require.NotNil(t, c2)
}
