// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package target

import (
	"context"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func newTimestampedOISnapshotFromProto(s *snapshot.TimestampedOpenInterest) timestampedOI {
	return timestampedOI{
		Time: time.Unix(0, s.Time),
		OI:   s.OpenInterest,
	}
}

func (toi timestampedOI) toSnapshotProto() *snapshot.TimestampedOpenInterest {
	return &snapshot.TimestampedOpenInterest{
		OpenInterest: toi.OI,
		Time:         toi.Time.UnixNano(),
	}
}

type SnapshotEngine struct {
	*Engine
	data    []byte
	stopped bool
	changed bool
	key     string
	keys    []string
}

func NewSnapshotEngine(
	parameters types.TargetStakeParameters,
	oiCalc OpenInterestCalculator,
	marketID string,
	positionFactor num.Decimal,
) *SnapshotEngine {
	key := (&types.PayloadLiquidityTarget{
		Target: &snapshot.LiquidityTarget{MarketId: marketID},
	}).Key()

	return &SnapshotEngine{
		Engine:  NewEngine(parameters, oiCalc, marketID, positionFactor),
		changed: true,
		key:     key,
		keys:    []string{key},
	}
}

func (e *SnapshotEngine) UpdateParameters(parameters types.TargetStakeParameters) {
	e.Engine.UpdateParameters(parameters)
}

func (e *SnapshotEngine) StopSnapshots() {
	e.stopped = true
}

func (e *SnapshotEngine) Changed() bool {
	return e.changed
}

func (e *SnapshotEngine) RecordOpenInterest(oi uint64, now time.Time) error {
	if err := e.Engine.RecordOpenInterest(oi, now); err != nil {
		return err
	}

	e.changed = true
	return nil
}

func (e *SnapshotEngine) GetTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint) *num.Uint {
	ts, changed := e.Engine.GetTargetStake(rf, now, markPrice)
	if changed {
		e.changed = true
	}
	return ts
}

func (e *SnapshotEngine) GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint, trades []*types.Trade) *num.Uint {
	tts, changed := e.Engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)
	if changed {
		e.changed = true
	}
	return tts
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.LiquidityTargetSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return e.keys
}

func (e *SnapshotEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshotEngine) HasChanged(k string) bool {
	return true
	// return e.changed
}

func (e *SnapshotEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != e.key {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, err := e.serialise()
	return state, nil, err
}

func (e *SnapshotEngine) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadLiquidityTarget:

		// Check the payload is for this market
		if e.marketID != pl.Target.MarketId {
			return nil, types.ErrUnknownSnapshotType
		}

		e.now = time.Unix(0, pl.Target.CurrentTime)
		e.scheduledTruncate = time.Unix(0, pl.Target.ScheduledTruncate)
		e.current = pl.Target.CurrentOpenInterests
		e.previous = make([]timestampedOI, 0, len(pl.Target.PreviousOpenInterests))
		e.max = newTimestampedOISnapshotFromProto(pl.Target.MaxOpenInterests)

		for _, poi := range pl.Target.PreviousOpenInterests {
			e.previous = append(e.previous, newTimestampedOISnapshotFromProto(poi))
		}

		var err error
		e.data, err = proto.Marshal(payload.IntoProto())
		e.changed = false
		return nil, err

	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) serialisePrevious() []*snapshot.TimestampedOpenInterest {
	poi := make([]*snapshot.TimestampedOpenInterest, 0, len(e.previous))
	for _, p := range e.previous {
		poi = append(poi, p.toSnapshotProto())
	}
	return poi
}

// serialise marshal the snapshot state, populating the data fields
// with updated values.
func (e *SnapshotEngine) serialise() ([]byte, error) {
	if e.stopped {
		return nil, nil
	}

	if !e.HasChanged(e.key) {
		return e.data, nil // we already have what we need
	}

	p := &snapshot.Payload{
		Data: &snapshot.Payload_LiquidityTarget{
			LiquidityTarget: &snapshot.LiquidityTarget{
				MarketId:              e.marketID,
				CurrentTime:           e.now.UnixNano(),
				ScheduledTruncate:     e.scheduledTruncate.UnixNano(),
				CurrentOpenInterests:  e.current,
				PreviousOpenInterests: e.serialisePrevious(),
				MaxOpenInterests:      e.max.toSnapshotProto(),
			},
		},
	}

	var err error
	e.data, err = proto.Marshal(p)
	if err != nil {
		return nil, err
	}

	e.changed = false

	return e.data, nil
}
