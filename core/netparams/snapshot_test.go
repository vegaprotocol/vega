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

package netparams_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetParamsSnapshots(t *testing.T) {
	t.Run("snapshot success", testSnapshotSuccess)
	t.Run("snapshot restore removed key", testSnapshotRestoreRemovedKey)
}

func testSnapshotSuccess(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	state, _, err := netp.GetState("all")
	require.NoError(t, err)
	assert.Greater(t, len(state), 0)

	// change a network parameter away from default
	netp.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	err = netp.Update(
		context.Background(), netparams.GovernanceProposalMarketMinClose, "1m")
	assert.NoError(t, err)

	nv, err := netp.Get(netparams.GovernanceProposalMarketMinClose)
	assert.NoError(t, err)
	assert.NotEmpty(t, nv)

	// check state is now different
	state2, _, err := netp.GetState("all")
	require.NoError(t, err)
	assert.NotEqual(t, state, state2)

	var pl snapshot.Payload
	proto.Unmarshal(state2, &pl)
	payload := types.PayloadFromProto(&pl)

	// load state
	snap := getTestNetParams(t)
	snap.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	_, err = snap.LoadState(context.Background(), payload)
	require.NoError(t, err)

	// check update value persists
	restored, err := snap.Get(netparams.GovernanceProposalMarketMinClose)
	fmt.Println(restored)
	require.NoError(t, err)
	assert.Equal(t, restored, nv)
}

func testSnapshotRestoreRemovedKey(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()

	// fiddle with the payload to add a nonsense network parameter to emulate a
	// deprecated parameter being removed over a PUP
	nps := &types.PayloadNetParams{
		NetParams: &types.NetParams{
			Params: []*types.NetworkParameter{
				{
					Key:   "hello",
					Value: "goodbye",
				},
			},
		},
	}

	pl := &types.Payload{
		Data: nps,
	}

	// load state
	_, err := netp.LoadState(context.Background(), pl)
	require.NoError(t, err)

	_, err = netp.Get("hello")
	assert.ErrorIs(t, err, netparams.ErrUnknownKey)
}
