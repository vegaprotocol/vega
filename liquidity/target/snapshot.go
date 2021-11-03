package target

import (
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
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
	hash    []byte
	data    []byte
	changed bool
	buf     *proto.Buffer
	key     string
	keys    []string
}

func NewSnapshotEngine(
	parameters types.TargetStakeParameters,
	oiCalc OpenInterestCalculator,
	marketID string,
) *SnapshotEngine {
	buf := proto.NewBuffer(nil)
	buf.SetDeterministic(true)

	key := (&types.PayloadLiquidityTarget{
		Target: &snapshot.LiquidityTarget{MarketId: marketID},
	}).Key()

	return &SnapshotEngine{
		Engine:  NewEngine(parameters, oiCalc, marketID),
		changed: true,
		buf:     buf,
		key:     key,
		keys:    []string{key},
	}
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
	return types.LiquiditySnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return e.keys
}

func (e *SnapshotEngine) GetHash(k string) ([]byte, error) {
	if k != e.key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	_, hash, err := e.serialise()
	return hash, err
}

func (e *SnapshotEngine) GetState(k string) ([]byte, error) {
	if k != e.key {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, _, err := e.serialise()
	return state, err
}

func (e *SnapshotEngine) LoadState(payload *types.Payload) error {
	if e.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadLiquidityTarget:

		// Check the payload is for this market
		if e.marketID != pl.Target.MarketId {
			return types.ErrUnknownSnapshotType
		}

		e.now = time.Unix(0, pl.Target.CurrentTime)
		e.scheduledTruncate = time.Unix(0, pl.Target.ScheduledTruncate)
		e.current = pl.Target.CurrentOpenInterests
		e.previous = make([]timestampedOI, 0, len(pl.Target.PreviousOpenInterests))
		e.max = newTimestampedOISnapshotFromProto(pl.Target.MaxOpenInterests)

		for _, poi := range pl.Target.PreviousOpenInterests {
			e.previous = append(e.previous, newTimestampedOISnapshotFromProto(poi))
		}

		e.changed = true
		return nil

	default:
		return types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) serialisePrevious() []*snapshot.TimestampedOpenInterest {
	poi := make([]*snapshot.TimestampedOpenInterest, 0, len(e.previous))
	for _, p := range e.previous {
		poi = append(poi, p.toSnapshotProto())
	}
	return poi
}

// serialise marshal the snapshot state, populating the data and hash fields
// with updated values.
func (e *SnapshotEngine) serialise() ([]byte, []byte, error) {
	if !e.changed {
		return e.data, e.hash, nil // we already have what we need
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

	e.buf.Reset()
	if err := e.buf.Marshal(p); err != nil {
		return nil, nil, err
	}

	e.data = e.buf.Bytes()
	e.hash = crypto.Hash(e.data)
	e.changed = false

	return e.data, e.hash, nil
}
