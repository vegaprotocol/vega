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

package spot

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func newTimestampedTotalStakeSnapshotFromProto(s *snapshot.TimestampedTotalStake) timestampedTotalStake {
	return timestampedTotalStake{
		Time:       time.Unix(0, s.Time),
		TotalStake: s.TotalStake,
	}
}

func (toi timestampedTotalStake) toSnapshotProto() *snapshot.TimestampedTotalStake {
	return &snapshot.TimestampedTotalStake{
		TotalStake: toi.TotalStake,
		Time:       toi.Time.UnixNano(),
	}
}

type SnapshotEngine struct {
	*Engine
	data    []byte
	stopped bool
	key     string
	keys    []string
}

func NewSnapshotEngine(
	parameters types.TargetStakeParameters,
	marketID string,
	positionFactor num.Decimal,
) *SnapshotEngine {
	key := (&types.PayloadSpotLiquidityTarget{
		Target: &snapshot.SpotLiquidityTarget{MarketId: marketID},
	}).Key()

	return &SnapshotEngine{
		Engine: NewEngine(parameters, marketID, positionFactor),
		key:    key,
		keys:   []string{key},
	}
}

func (e *SnapshotEngine) UpdateParameters(parameters types.TargetStakeParameters) {
	e.Engine.UpdateParameters(parameters)
}

func (e *SnapshotEngine) StopSnapshots() {
	e.stopped = true
}

func (e *SnapshotEngine) RecordTotalStake(oi uint64, now time.Time) error {
	if err := e.Engine.RecordTotalStake(oi, now); err != nil {
		return err
	}

	return nil
}

func (e *SnapshotEngine) GetTargetStake(now time.Time) *num.Uint {
	return e.Engine.GetTargetStake(now)
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
	case *types.PayloadSpotLiquidityTarget:

		// Check the payload is for this market
		if e.marketID != pl.Target.MarketId {
			return nil, types.ErrUnknownSnapshotType
		}

		e.now = time.Unix(0, pl.Target.CurrentTime)
		e.scheduledTruncate = time.Unix(0, pl.Target.ScheduledTruncate)
		e.current = pl.Target.CurrentTotalStake
		e.previous = make([]timestampedTotalStake, 0, len(pl.Target.PreviousTotalStake))
		e.max = newTimestampedTotalStakeSnapshotFromProto(pl.Target.MaxTotalStake)

		for _, poi := range pl.Target.PreviousTotalStake {
			e.previous = append(e.previous, newTimestampedTotalStakeSnapshotFromProto(poi))
		}

		var err error
		e.data, err = proto.Marshal(payload.IntoProto())
		return nil, err

	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) serialisePrevious() []*snapshot.TimestampedTotalStake {
	poi := make([]*snapshot.TimestampedTotalStake, 0, len(e.previous))
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

	p := &snapshot.Payload{
		Data: &snapshot.Payload_SpotLiquidityTarget{
			SpotLiquidityTarget: &snapshot.SpotLiquidityTarget{
				MarketId:           e.marketID,
				CurrentTime:        e.now.UnixNano(),
				ScheduledTruncate:  e.scheduledTruncate.UnixNano(),
				CurrentTotalStake:  e.current,
				PreviousTotalStake: e.serialisePrevious(),
				MaxTotalStake:      e.max.toSnapshotProto(),
			},
		},
	}

	var err error
	e.data, err = proto.Marshal(p)
	if err != nil {
		return nil, err
	}
	return e.data, nil
}
