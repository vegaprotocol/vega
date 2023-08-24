// Copyright (c) 2023 Gobalsky Labs Limited
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

package referral_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTakingAndRestoringSnapshotSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	vegaPath := paths.New(t.TempDir())
	now := time.Now()

	te1 := newEngine(t)
	snapshotEngine1 := newSnapshotEngine(t, vegaPath, now, te1.engine)
	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	require.NoError(t, snapshotEngine1.Start(ctx))

	program1 := &types.ReferralProgram{
		EndOfProgramTimestamp: time.Now().Add(24 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	te1.engine.UpdateProgram(program1)

	// Simulating end of epoch.
	// The program should be applied.
	expectReferralProgramStartedEvent(t, te1)
	lastEpochStartTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te1, lastEpochStartTime)

	program2 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochStartTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Set new program.
	te1.engine.UpdateProgram(program2)

	// Take a snapshot.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	// Simulating end of epoch.
	// The program should be updated with the new one.
	expectReferralProgramUpdatedEvent(t, te1)
	lastEpochStartTime = program2.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te1, lastEpochStartTime)

	state1 := map[string][]byte{}
	for _, key := range te1.engine.Keys() {
		state, additionalProvider, err := te1.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	closeSnapshotEngine1()

	// Reload the engine using the previous snapshot.

	te2 := newEngine(t)
	snapshotEngine2 := newSnapshotEngine(t, vegaPath, now, te2.engine)
	defer snapshotEngine2.Close()

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	// Simulating end of epoch.
	// The program should be updated with the new one.
	expectReferralProgramUpdatedEvent(t, te2)
	lastEpochStartTime = program2.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te2, lastEpochStartTime)

	state2 := map[string][]byte{}
	for _, key := range te2.engine.Keys() {
		state, additionalProvider, err := te2.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
