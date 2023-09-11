package liquidity

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type snapshotV1 struct {
	*Engine
	market  string
	stopped bool
	keys    []string
}

func (e *snapshotV1) Namespace() types.SnapshotNamespace {
	return types.LiquiditySnapshot
}

func (e *snapshotV1) Keys() []string {
	if len(e.keys) <= 0 {
		e.keys = []string{
			(&types.PayloadLiquidityParameters{
				Parameters: &snapshotpb.LiquidityParameters{
					MarketId: e.market,
				},
			}).Key(),
			(&types.PayloadLiquidityPendingProvisions{
				PendingProvisions: &snapshotpb.LiquidityPendingProvisions{
					MarketId: e.market,
				},
			}).Key(),
			(&types.PayloadLiquidityProvisions{
				Provisions: &snapshotpb.LiquidityProvisions{
					MarketId: e.market,
				},
			}).Key(),
			(&types.PayloadLiquiditySupplied{
				LiquiditySupplied: &snapshotpb.LiquiditySupplied{
					MarketId: e.market,
				},
			}).Key(),
			(&types.PayloadLiquidityScores{
				LiquidityScores: &snapshotpb.LiquidityScores{
					MarketId: e.market,
				},
			}).Key(),
		}
	}

	return e.keys
}

func (e *snapshotV1) GetState(k string) ([]byte, []types.StateProvider, error) {
	var (
		keys  = e.Keys()
		state = &snapshotpb.Payload{}
	)

	switch k {
	case keys[0]:
		state.Data = &snapshotpb.Payload_LiquidityParameters{
			LiquidityParameters: &snapshotpb.LiquidityParameters{
				MarketId: e.market,
			},
		}
	case keys[1]:
		state.Data = &snapshotpb.Payload_LiquidityPendingProvisions{
			LiquidityPendingProvisions: &snapshotpb.LiquidityPendingProvisions{
				MarketId: e.market,
			},
		}
	case keys[2]:
		state.Data = &snapshotpb.Payload_LiquidityProvisions{
			LiquidityProvisions: &snapshotpb.LiquidityProvisions{
				MarketId: e.market,
			},
		}
	case keys[3]:
		state.Data = &snapshotpb.Payload_LiquiditySupplied{
			LiquiditySupplied: &snapshotpb.LiquiditySupplied{
				MarketId: e.market,
			},
		}
	case keys[4]:
		state.Data = &snapshotpb.Payload_LiquidityScores{
			LiquidityScores: &snapshotpb.LiquidityScores{
				MarketId: e.market,
			},
		}
	}

	buf, err := proto.Marshal(state)
	if err != nil {
		return nil, nil, err
	}

	return buf, nil, nil
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

		e.loadPerformances(pl.Provisions.GetLiquidityProvisions())
		return nil, nil
	case *types.PayloadLiquidityScores:
		return nil, e.loadScores(pl.LiquidityScores)
	case *types.PayloadLiquiditySupplied:
		return nil, e.loadSupplied(pl.LiquiditySupplied)
	case *types.PayloadLiquidityPendingProvisions:
		return nil, nil
	case *types.PayloadLiquidityParameters:
		return nil, nil

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

func (e *snapshotV1) loadPerformances(provisions []*typespb.LiquidityProvision) {
	e.slaPerformance = map[string]*slaPerformance{}
	for _, provision := range provisions {
		previousPenalties := restoreSliceRing(
			[]*num.Decimal{},
			e.slaParams.PerformanceHysteresisEpochs,
			0,
		)

		var startTime time.Time

		e.slaPerformance[provision.PartyId] = &slaPerformance{
			s:                 0,
			start:             startTime,
			previousPenalties: previousPenalties,
		}
	}
}

func (e *snapshotV1) loadProvisions(ctx context.Context, provisions []*typespb.LiquidityProvision) error {
	e.provisions = newSnapshotableProvisionsPerParty()

	evts := make([]events.Event, 0, len(provisions))
	for _, v := range provisions {
		provision, err := types.LiquidityProvisionFromProto(v)
		if err != nil {
			return err
		}
		e.provisions.Set(v.PartyId, provision)
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
