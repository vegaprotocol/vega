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

package limits_test

import (
	"bytes"
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/limits"
	"code.vegaprotocol.io/vega/core/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var allKey = (&types.PayloadLimitState{}).Key()

func TestLimitSnapshotEmpty(t *testing.T) {
	l := getLimitsTest(t)

	s, _, err := l.GetState(allKey)
	require.Nil(t, err)
	require.NotNil(t, s)
}

func TestLimitSnapshotWrongPayLoad(t *testing.T) {
	l := getLimitsTest(t)
	snap := &types.Payload{Data: &types.PayloadEpoch{}}
	_, err := l.LoadState(context.Background(), snap)
	assert.ErrorIs(t, types.ErrInvalidSnapshotNamespace, err)
}

func TestLimitSnapshotGenesisState(t *testing.T) {
	gs := &limits.GenesisState{}
	lmt := getLimitsTest(t)
	s1, _, err := lmt.GetState(allKey)
	require.Nil(t, err)

	lmt.loadGenesisState(t, gs)

	s2, _, err := lmt.GetState(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))
}

func TestSnapshotRoundTrip(t *testing.T) {
	gs := &limits.GenesisState{}
	lmt := getLimitsTest(t)
	lmt.loadGenesisState(t, gs)
	lmt.OnLimitsProposeSpotMarketEnabledFromUpdate(context.Background(), 1)

	s1, _, err := lmt.GetState(allKey)
	require.Nil(t, err)

	s2, _, err := lmt.GetState(allKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s1, s2))
}
