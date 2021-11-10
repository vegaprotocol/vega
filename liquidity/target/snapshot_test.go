package target_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/liquidity/target"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func newSnapshotEngine(marketID string) *target.SnapshotEngine {
	params := types.TargetStakeParameters{
		TimeWindow:    5,
		ScalingFactor: num.NewDecimalFromFloat(2),
	}
	var oiCalc target.OpenInterestCalculator

	return target.NewSnapshotEngine(params, oiCalc, marketID)
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	a := assert.New(t)
	marketID := "market-1"
	key := fmt.Sprintf("target:%s", marketID)
	se := newSnapshotEngine(marketID)

	h, err := se.GetHash("")
	a.Empty(h)
	a.EqualError(err, types.ErrSnapshotKeyDoesNotExist.Error())
	h, _, err = se.GetState("")
	a.Empty(h)
	a.EqualError(err, types.ErrSnapshotKeyDoesNotExist.Error())

	h, err = se.GetHash(key)
	a.NotEmpty(h)
	a.NoError(err)

	d := time.Date(2015, time.December, 24, 19, 0, 0, 0, time.UTC)
	se.RecordOpenInterest(40, d)
	se.RecordOpenInterest(40, d.Add(time.Hour*3))

	h1, err := se.GetHash(key)
	a.NotEmpty(h1)
	a.NoError(err)
	s, _, err := se.GetState(key)
	a.NotEmpty(s)
	a.NoError(err)

	se2 := newSnapshotEngine(marketID)

	pl := snapshot.Payload{}
	assert.NoError(t, proto.Unmarshal(s, &pl))

	_, err = se2.LoadState(context.TODO(), types.PayloadFromProto(&pl))
	a.NoError(err)

	h2, err := se2.GetHash(key)
	a.NoError(err)
	a.Equal(h1, h2)
}
