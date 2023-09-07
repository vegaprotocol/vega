package liquidity

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type snapshotV1 struct {
	*Engine

	stopped  bool
	hashKeys []string
}

func (e *snapshotV1) Namespace() types.SnapshotNamespace {
	return types.LiquiditySnapshot
}

func (e *snapshotV1) Keys() []string {
	return e.hashKeys
}

func (e *snapshotV1) GetState(k string) ([]byte, []types.StateProvider, error) {
	return nil, nil, nil
}

func (e *snapshotV1) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := p.Data.(type) {
	case *types.PayloadLiquidityProvisions:
		if err := e.loadProvisions(ctx, pl.Provisions.GetLiquidityProvisions()); err != nil {
			return nil, err
		}

		return nil, e.loadPerformances(pl.Provisions.GetLiquidityProvisions())
	case *types.PayloadLiquidityScores:
		return nil, e.loadScores(pl.LiquidityScores)
	case *types.PayloadLiquiditySupplied:
		return nil, e.loadSupplied(pl.LiquiditySupplied)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *snapshotV1) Stopped() bool {
	return e.stopped
}

func (e *snapshotV1) Stop() {
	e.log.Debug("market has been cleared, stopping snapshot production", logging.MarketID(e.marketID))
	e.stopped = true
}

func (e *snapshotV1) loadPerformances(provisions []*typespb.LiquidityProvision) error {
	var err error

	// TODO karel - how to get the time?
	// e.Engine.slaEpochStart = time.Unix(0, performances.EpochStartTime)

	e.Engine.slaPerformance = map[string]*slaPerformance{}
	for _, provision := range provisions {
		previousPenalties := restoreSliceRing[*num.Decimal](
			[]*num.Decimal{},
			e.Engine.slaParams.PerformanceHysteresisEpochs,
			0,
		)

		var startTime time.Time

		e.Engine.slaPerformance[provision.PartyId] = &slaPerformance{
			s:                 0,
			start:             startTime,
			previousPenalties: previousPenalties,
		}
	}

	return err
}

func (e *snapshotV1) loadProvisions(ctx context.Context, provisions []*typespb.LiquidityProvision) error {
	e.Engine.provisions = newSnapshotableProvisionsPerParty()

	evts := make([]events.Event, 0, len(provisions))
	for _, v := range provisions {
		provision, err := types.LiquidityProvisionFromProto(v)
		if err != nil {
			return err
		}
		e.Engine.provisions.Set(v.PartyId, provision)
		evts = append(evts, events.NewLiquidityProvisionEvent(ctx, provision))
	}

	var err error
	e.broker.SendBatch(evts)
	return err
}

func (e *snapshotV1) loadSupplied(ls *snapshotpb.LiquiditySupplied) error {
	// Dirty hack so we can reuse the supplied engine from the liquidity engine v1,
	// without snapshot payload namespace issue.
	err := e.suppliedEngine.Reload(&snapshotpb.LiquiditySupplied{
		MarketId:         ls.MarketId,
		ConsensusReached: ls.ConsensusReached,
		BidCache:         ls.BidCache,
		AskCache:         ls.AskCache,
	})
	if err != nil {
		return err
	}

	return err
}

func (e *snapshotV1) loadScores(ls *snapshotpb.LiquidityScores) error {
	var err error

	e.nAvg = int64(ls.RunningAverageCounter)

	scores := make(map[string]num.Decimal, len(ls.Scores))
	for _, p := range ls.Scores {
		score, err := num.DecimalFromString(p.Score)
		if err != nil {
			return err
		}
		scores[p.PartyId] = score
	}

	e.avgScores = scores
	return err
}
