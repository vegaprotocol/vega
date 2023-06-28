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
	"testing"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRestoreDependentNetparams(t *testing.T) {
	netp := getTestNetParams(t)
	defer netp.ctrl.Finish()
	ctx := context.Background()

	netp.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// get the original default value
	err := netp.Update(
		context.Background(), netparams.MarketAuctionMinimumDuration, "1s")
	assert.NoError(t, err)

	// now change max
	err = netp.Update(
		context.Background(), netparams.MarketAuctionMaximumDuration, "10s")
	assert.NoError(t, err)

	// now snapshot restore
	data, _, err := netp.GetState("all")
	require.NoError(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data, snap)
	require.Nil(t, err)

	snapNetp := getTestNetParams(t)
	snapNetp.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	_, err = snapNetp.LoadState(ctx, types.PayloadFromProto(snap))
	require.NoError(t, err)

	v1, err := snapNetp.Get(netparams.MarketAuctionMaximumDuration)
	require.NoError(t, err)
	v2, err := netp.Get(netparams.MarketAuctionMaximumDuration)
	require.NoError(t, err)

	require.Equal(t, v1, v2)
}
