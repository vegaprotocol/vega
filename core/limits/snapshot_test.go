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
