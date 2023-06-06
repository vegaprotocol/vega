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

package spot_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/core/liquidity/target/spot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
)

func newSnapshotEngine(marketID string) *spot.SnapshotEngine {
	params := types.TargetStakeParameters{
		TimeWindow:    5,
		ScalingFactor: num.NewDecimalFromFloat(2),
	}

	return spot.NewSnapshotEngine(params, marketID, num.DecimalFromFloat(1))
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	a := assert.New(t)
	marketID := "market-1"
	key := fmt.Sprintf("target:%s", marketID)
	se := newSnapshotEngine(marketID)

	s, _, err := se.GetState("")
	a.Empty(s)
	a.EqualError(err, types.ErrSnapshotKeyDoesNotExist.Error())

	d := time.Date(2015, time.December, 24, 19, 0, 0, 0, time.UTC)
	se.RecordTotalStake(40, d)
	se.RecordTotalStake(40, d.Add(time.Hour*3))

	s, _, err = se.GetState(key)
	a.NotEmpty(s)
	a.NoError(err)

	se2 := newSnapshotEngine(marketID)

	pl := snapshot.Payload{}
	assert.NoError(t, proto.Unmarshal(s, &pl))

	_, err = se2.LoadState(context.TODO(), types.PayloadFromProto(&pl))
	a.NoError(err)

	s2, _, err := se2.GetState(key)
	a.NoError(err)
	a.True(bytes.Equal(s, s2))
}

func TestStopSnapshotTaking(t *testing.T) {
	marketID := "market-1"
	key := fmt.Sprintf("target:%s", marketID)
	se := newSnapshotEngine(marketID)

	// signal to kill the engine's snapshots
	se.StopSnapshots()

	s, _, err := se.GetState(key)
	assert.NoError(t, err)
	assert.Nil(t, s)
	assert.True(t, se.Stopped())
}
