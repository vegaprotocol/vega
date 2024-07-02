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

package parties_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

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

	party1 := newPartyID(t)

	expectPartyProfileUpdatedEvent(t, te1)
	require.NoError(t, te1.engine.UpdateProfile(ctx, party1, &commandspb.UpdatePartyProfile{
		Alias: "test1",
		Metadata: []*vegapb.Metadata{
			{
				Key:   "key1",
				Value: "value1",
			},
		},
	}))

	party2 := newPartyID(t)

	expectPartyProfileUpdatedEvent(t, te1)
	require.NoError(t, te1.engine.UpdateProfile(ctx, party2, &commandspb.UpdatePartyProfile{
		Alias: "test2",
		Metadata: []*vegapb.Metadata{
			{
				Key:   "key1",
				Value: "value1",
			},
		},
	}))

	// Take a snapshot.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	party3 := newPartyID(t)

	postSnapshot := func(te *testEngine) {
		expectPartyProfileUpdatedEvent(t, te)
		require.NoError(t, te.engine.UpdateProfile(ctx, party3, &commandspb.UpdatePartyProfile{
			Alias: "test3",
			Metadata: []*vegapb.Metadata{
				{
					Key:   "key1",
					Value: "value1",
				},
			},
		}))

		assertEqualProfiles(t, []types.PartyProfile{
			{
				PartyID: party1,
				Alias:   "test1",
				Metadata: map[string]string{
					"key1": "value1",
				},
				DerivedKeys: map[string]struct{}{},
			},
			{
				PartyID: party2,
				Alias:   "test2",
				Metadata: map[string]string{
					"key1": "value1",
				},
				DerivedKeys: map[string]struct{}{},
			},
			{
				PartyID: party3,
				Alias:   "test3",
				Metadata: map[string]string{
					"key1": "value1",
				},
				DerivedKeys: map[string]struct{}{},
			},
		}, te.engine.ListProfiles())
	}

	postSnapshot(te1)

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

	// Re-applying exact same steps after the snapshot is taken to see if it leads
	// to the same state.
	postSnapshot(te2)

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
