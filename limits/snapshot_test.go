package limits_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var allKey = (&types.PayloadLimitState{}).Key()

func TestLimitSnapshotEmpty(t *testing.T) {
	l := getLimitsTest(t)

	h, err := l.GetHash(allKey)
	require.Nil(t, err)
	require.NotNil(t, h)
}

func TestLimitSnapshotWrongPayLoad(t *testing.T) {
	l := getLimitsTest(t)
	snap := &types.Payload{Data: &types.PayloadEpoch{}}
	_, err := l.LoadState(context.Background(), snap)
	assert.ErrorIs(t, types.ErrInvalidSnapshotNamespace, err)
}

func TestLimitSnapshotGenesisState(t *testing.T) {
	gs := &limits.GenesisState{
		BootstrapBlockCount: 1,
	}
	lmt := getLimitsTest(t)
	h1, err := lmt.GetHash(allKey)
	require.Nil(t, err)

	lmt.loadGenesisState(t, gs)

	h2, err := lmt.GetHash(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
}

func TestLimitSnapshotBlockCount(t *testing.T) {
	ctx := context.Background()
	gs := &limits.GenesisState{
		BootstrapBlockCount: 1,
	}
	lmt := getLimitsTest(t)
	lmt.loadGenesisState(t, gs)

	h1, err := lmt.GetHash(allKey)
	require.Nil(t, err)

	// increase block count and hash should change
	lmt.OnTick(ctx, time.Unix(3000, 0))
	require.False(t, lmt.BootstrapFinished())

	h2, err := lmt.GetHash(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))

	state, _, err := lmt.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	// Load state into new engine and check the blockcount has returned
	// be counting the expected steps for boostrapping to have finished
	snapLmt := getLimitsTest(t)
	snapLmt.loadGenesisState(t, gs)
	_, err = snapLmt.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)
	require.False(t, snapLmt.BootstrapFinished())

	snapLmt.OnTick(context.Background(), time.Unix(4000, 0))
	require.True(t, snapLmt.BootstrapFinished())
}

func TestLimitSnapshotBootstrapFinished(t *testing.T) {
	ctx := context.Background()
	gs := &limits.GenesisState{
		BootstrapBlockCount:  0,
		ProposeMarketEnabled: true,
		ProposeAssetEnabled:  true,
	}
	lmt := getLimitsTest(t)
	lmt.loadGenesisState(t, gs)

	// Tick to get out of bootstrapping
	lmt.OnTick(ctx, time.Unix(3000, 0))
	require.True(t, lmt.CanProposeAsset())
	require.True(t, lmt.CanProposeMarket())
	require.True(t, lmt.BootstrapFinished())

	state, _, err := lmt.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	// Load state into new engine and check all the flags have returned
	snapLmt := getLimitsTest(t)
	snapLmt.loadGenesisState(t, gs)
	_, err = snapLmt.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)
	require.True(t, lmt.CanProposeAsset())
	require.True(t, lmt.CanProposeMarket())
	require.True(t, lmt.BootstrapFinished())

	h1, err := lmt.GetHash(allKey)
	require.Nil(t, err)
	h2, err := snapLmt.GetHash(allKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(h1, h2))
}
