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

	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/require"
)

func TestUpdatingProfiles(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	party1 := newPartyID(t)

	expectPartyProfileUpdatedEvent(t, te)
	require.NoError(t, te.engine.UpdateProfile(ctx, party1, &commandspb.UpdatePartyProfile{
		Alias: "test1",
		Metadata: []*vegapb.Metadata{
			{
				Key:   "key1",
				Value: "value1",
			},
		},
	}))

	party2 := newPartyID(t)

	expectPartyProfileUpdatedEvent(t, te)
	require.NoError(t, te.engine.UpdateProfile(ctx, party2, &commandspb.UpdatePartyProfile{
		Alias: "test2",
		Metadata: []*vegapb.Metadata{
			{
				Key:   "key1",
				Value: "value1",
			},
		},
	}))

	expectPartyProfileUpdatedEvent(t, te)
	require.NoError(t, te.engine.UpdateProfile(ctx, party1, &commandspb.UpdatePartyProfile{
		Alias: "test1",
		Metadata: []*vegapb.Metadata{
			{
				Key:   "key2",
				Value: "value2",
			},
			{
				Key:   "key3",
				Value: "value3",
			},
		},
	}))

	// Attempt using alias from party 2.
	require.Error(t, te.engine.UpdateProfile(ctx, party1, &commandspb.UpdatePartyProfile{
		Alias: "test2",
	}))

	assertEqualProfiles(t, []types.PartyProfile{
		{
			PartyID: party1,
			Alias:   "test1",
			Metadata: map[string]string{
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			PartyID: party2,
			Alias:   "test2",
			Metadata: map[string]string{
				"key1": "value1",
			},
		},
	}, te.engine.ListProfiles())
}
