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

package target_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/core/liquidity/target"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
)

func newSnapshotEngine(marketID string) *target.SnapshotEngine {
	params := types.TargetStakeParameters{
		TimeWindow:    5,
		ScalingFactor: num.NewDecimalFromFloat(2),
	}
	var oiCalc target.OpenInterestCalculator

	return target.NewSnapshotEngine(params, oiCalc, marketID, num.DecimalFromFloat(1))
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
	se.RecordOpenInterest(40, d)
	se.RecordOpenInterest(40, d.Add(time.Hour*3))

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
