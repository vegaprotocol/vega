package liquidity

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type SnapshotEngine struct {
	*Engine

	*snapshotV2
	*snapshotV1
}

func (e SnapshotEngine) V2StateProvider() types.StateProvider {
	return e.snapshotV2
}

func (e SnapshotEngine) V1StateProvider() types.StateProvider {
	return e.snapshotV1
}

func (e SnapshotEngine) StopSnapshots() {
	e.snapshotV1.Stop()
	e.snapshotV2.Stop()
}

type snapshotV2 struct {
	*Engine

	pl     types.Payload
	market string

	// liquidity types
	stopped                     bool
	serialisedProvisions        []byte
	serialisedPendingProvisions []byte
	serialisedPerformances      []byte
	serialisedSupplied          []byte
	serialisedScores            []byte

	// Keys need to be computed when the engine is instantiated as they are dynamic.
	hashKeys             []string
	provisionsKey        string
	pendingProvisionsKey string
	performancesKey      string
	scoresKey            string
	suppliedKey          string
}

func (e *snapshotV2) Namespace() types.SnapshotNamespace {
	return types.LiquidityV2Snapshot
}

func (e *snapshotV2) Keys() []string {
	return e.hashKeys
}

func (e *snapshotV2) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *snapshotV2) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := p.Data.(type) {
	case *types.PayloadLiquidityV2Provisions:
		return nil, e.loadProvisions(ctx, pl.Provisions.GetLiquidityProvisions(), p)
	case *types.PayloadLiquidityV2PendingProvisions:
		return nil, e.loadPendingProvisions(ctx, pl.PendingProvisions.GetPendingLiquidityProvisions(), p)
	case *types.PayloadLiquidityV2Performances:
		return nil, e.loadPerformances(pl.Performances, p)
	case *types.PayloadLiquidityV2Supplied:
		return nil, e.loadSupplied(pl.Supplied, p)
	case *types.PayloadLiquidityV2Scores:
		return nil, e.loadScores(pl.Scores, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *snapshotV2) Stopped() bool {
	return e.stopped
}

func (e *snapshotV2) Stop() {
	e.log.Debug("market has been cleared, stopping snapshot production", logging.MarketID(e.marketID))
	e.stopped = true
}

func (e *snapshotV2) serialise(k string) ([]byte, error) {
	var (
		buf []byte
		err error
	)

	switch k {
	case e.provisionsKey:
		buf, err = e.serialiseProvisions()
	case e.pendingProvisionsKey:
		buf, err = e.serialisePendingProvisions()
	case e.performancesKey:
		buf, err = e.serialisePerformances()
	case e.suppliedKey:
		buf, err = e.serialiseSupplied()
	case e.scoresKey:
		buf, err = e.serialiseScores()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if err != nil {
		return nil, err
	}

	if e.stopped {
		return nil, nil
	}

	switch k {
	case e.provisionsKey:
		e.serialisedProvisions = buf
	case e.pendingProvisionsKey:
		e.serialisedPendingProvisions = buf
	case e.performancesKey:
		e.serialisedPerformances = buf
	case e.suppliedKey:
		e.serialisedSupplied = buf
	case e.scoresKey:
		e.serialisedScores = buf
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	return buf, nil
}

func (e *snapshotV2) serialiseProvisions() ([]byte, error) {
	// these are sorted already, only a conversion to proto is needed
	lps := e.Engine.provisions.Slice()
	pblps := make([]*typespb.LiquidityProvision, 0, len(lps))
	for _, v := range lps {
		pblps = append(pblps, v.IntoProto())
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityV2Provisions{
			LiquidityV2Provisions: &snapshotpb.LiquidityV2Provisions{
				MarketId:            e.market,
				LiquidityProvisions: pblps,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *snapshotV2) serialisePendingProvisions() ([]byte, error) {
	// these are sorted already, only a conversion to proto is needed
	lps := e.Engine.pendingProvisions.Slice()
	pblps := make([]*typespb.LiquidityProvision, 0, len(lps))
	for _, v := range lps {
		pblps = append(pblps, v.IntoProto())
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityV2PendingProvisions{
			LiquidityV2PendingProvisions: &snapshotpb.LiquidityV2PendingProvisions{
				MarketId:                   e.market,
				PendingLiquidityProvisions: pblps,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *snapshotV2) serialisePerformances() ([]byte, error) {
	// Extract and sort the parties to serialize a deterministic array.
	parties := make([]string, 0, len(e.slaPerformance))
	for party := range e.slaPerformance {
		parties = append(parties, party)
	}
	sort.Strings(parties)

	performancePerPartySnapshot := make([]*snapshotpb.LiquidityV2PerformancePerParty, 0, len(e.slaPerformance))
	for _, party := range parties {
		partyPerformance := e.slaPerformance[party]

		trueLen := 0
		registeredPenaltiesPerEpochSnapshot := make([]string, 0, partyPerformance.previousPenalties.Len())
		for _, registeredPenalty := range partyPerformance.previousPenalties.Slice() {
			if registeredPenalty != nil {
				trueLen++
				registeredPenaltiesPerEpochSnapshot = append(registeredPenaltiesPerEpochSnapshot, registeredPenalty.String())
			}
		}
		registeredPenaltiesPerEpochSnapshot = registeredPenaltiesPerEpochSnapshot[0:trueLen]

		var start int64
		if partyPerformance.start != (time.Time{}) {
			start = partyPerformance.start.UnixNano()
		}

		partyPerformanceSnapshot := &snapshotpb.LiquidityV2PerformancePerParty{
			Party:                            party,
			ElapsedTimeMeetingSlaDuringEpoch: int64(partyPerformance.s),
			CommitmentStartTime:              start,
			RegisteredPenaltiesPerEpoch:      registeredPenaltiesPerEpochSnapshot,
			PositionInPenaltiesPerEpoch:      uint32(partyPerformance.previousPenalties.Position()),
		}

		performancePerPartySnapshot = append(performancePerPartySnapshot, partyPerformanceSnapshot)
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityV2Performances{
			LiquidityV2Performances: &snapshotpb.LiquidityV2Performances{
				MarketId:            e.market,
				EpochStartTime:      e.slaEpochStart.UnixNano(),
				PerformancePerParty: performancePerPartySnapshot,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *snapshotV2) serialiseSupplied() ([]byte, error) {
	v1Payload := e.suppliedEngine.Payload()

	// Dirty hack to support serialization of a mutualized supplied engine between
	// liquidity engine version 1 and 2.
	supplied := v1Payload.GetLiquiditySupplied()
	return e.marshalPayload(&snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityV2Supplied{
			LiquidityV2Supplied: &snapshotpb.LiquidityV2Supplied{
				MarketId:         supplied.MarketId,
				ConsensusReached: supplied.ConsensusReached,
				BidCache:         supplied.BidCache,
				AskCache:         supplied.AskCache,
			},
		},
	})
}

func (e *snapshotV2) serialiseScores() ([]byte, error) {
	scores := make([]*snapshotpb.LiquidityScore, 0, len(e.avgScores))

	keys := make([]string, 0, len(e.avgScores))
	for k := range e.avgScores {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		s := &snapshotpb.LiquidityScore{
			PartyId: k,
			Score:   e.avgScores[k].String(),
		}
		scores = append(scores, s)
	}

	var lastFeeDistributionTime int64
	if e.lastFeeDistribution != (time.Time{}) {
		lastFeeDistributionTime = e.lastFeeDistribution.UnixNano()
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquidityV2Scores{
			LiquidityV2Scores: &snapshotpb.LiquidityV2Scores{
				MarketId:                e.market,
				RunningAverageCounter:   int32(e.nAvg),
				Scores:                  scores,
				LastFeeDistributionTime: lastFeeDistributionTime,
			},
		},
	}

	return e.marshalPayload(payload)
}

func (e *snapshotV2) marshalPayload(payload *snapshotpb.Payload) ([]byte, error) {
	buf, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (e *snapshotV2) loadProvisions(ctx context.Context, provisions []*typespb.LiquidityProvision, p *types.Payload) error {
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
	e.serialisedProvisions, err = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	return err
}

func (e *snapshotV2) loadPendingProvisions(ctx context.Context, provisions []*typespb.LiquidityProvision, p *types.Payload) error {
	e.Engine.pendingProvisions = newSnapshotablePendingProvisions()

	evts := make([]events.Event, 0, len(provisions))
	for _, v := range provisions {
		provision, err := types.LiquidityProvisionFromProto(v)
		if err != nil {
			return err
		}
		e.Engine.pendingProvisions.Set(provision)
		evts = append(evts, events.NewLiquidityProvisionEvent(ctx, provision))
	}

	var err error
	e.serialisedPendingProvisions, err = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	return err
}

func (e *snapshotV2) loadPerformances(performances *snapshotpb.LiquidityV2Performances, p *types.Payload) error {
	var err error

	e.Engine.slaEpochStart = time.Unix(0, performances.EpochStartTime)

	e.Engine.slaPerformance = map[string]*slaPerformance{}
	for _, partyPerformance := range performances.PerformancePerParty {
		registeredPenaltiesPerEpochAsDecimal := make([]*num.Decimal, 0, len(partyPerformance.RegisteredPenaltiesPerEpoch))
		for _, registeredPenalty := range partyPerformance.RegisteredPenaltiesPerEpoch {
			registeredPenaltyAsDecimal, err := num.DecimalFromString(registeredPenalty)
			if err != nil {
				return fmt.Errorf("invalid penalty %q for party %q on market %q: %w", registeredPenalty, partyPerformance.Party, performances.MarketId, err)
			}
			registeredPenaltiesPerEpochAsDecimal = append(registeredPenaltiesPerEpochAsDecimal, &registeredPenaltyAsDecimal)
		}

		previousPenalties := restoreSliceRing[*num.Decimal](
			registeredPenaltiesPerEpochAsDecimal,
			e.Engine.slaParams.PerformanceHysteresisEpochs,
			int(partyPerformance.PositionInPenaltiesPerEpoch),
		)

		var startTime time.Time
		if partyPerformance.CommitmentStartTime > 0 {
			startTime = time.Unix(0, partyPerformance.CommitmentStartTime)
		}

		e.Engine.slaPerformance[partyPerformance.Party] = &slaPerformance{
			s:                 time.Duration(partyPerformance.ElapsedTimeMeetingSlaDuringEpoch),
			start:             startTime,
			previousPenalties: previousPenalties,
		}
	}

	e.serialisedPerformances, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *snapshotV2) loadSupplied(ls *snapshotpb.LiquidityV2Supplied, p *types.Payload) error {
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
	e.serialisedSupplied, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *snapshotV2) loadScores(ls *snapshotpb.LiquidityV2Scores, p *types.Payload) error {
	var err error

	e.nAvg = int64(ls.RunningAverageCounter)
	e.lastFeeDistribution = time.Unix(0, ls.LastFeeDistributionTime)

	scores := make(map[string]num.Decimal, len(ls.Scores))
	for _, p := range ls.Scores {
		score, err := num.DecimalFromString(p.Score)
		if err != nil {
			return err
		}
		scores[p.PartyId] = score
	}

	e.avgScores = scores

	e.serialisedScores, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *snapshotV2) buildHashKeys(market string) {
	e.provisionsKey = (&types.PayloadLiquidityV2Provisions{
		Provisions: &snapshotpb.LiquidityV2Provisions{
			MarketId: market,
		},
	}).Key()

	e.pendingProvisionsKey = (&types.PayloadLiquidityV2PendingProvisions{
		PendingProvisions: &snapshotpb.LiquidityV2PendingProvisions{
			MarketId: market,
		},
	}).Key()

	e.performancesKey = (&types.PayloadLiquidityV2Performances{
		Performances: &snapshotpb.LiquidityV2Performances{
			MarketId: market,
		},
	}).Key()

	e.suppliedKey = (&types.PayloadLiquidityV2Supplied{
		Supplied: &snapshotpb.LiquidityV2Supplied{
			MarketId: market,
		},
	}).Key()

	e.scoresKey = (&types.PayloadLiquidityV2Scores{
		Scores: &snapshotpb.LiquidityV2Scores{
			MarketId: market,
		},
	}).Key()

	e.hashKeys = append([]string{},
		e.provisionsKey,
		e.pendingProvisionsKey,
		e.performancesKey,
		e.suppliedKey,
		e.scoresKey,
	)
}

func defaultLiquiditySLAParams() *types.LiquiditySLAParams {
	return &types.LiquiditySLAParams{
		PriceRange:                  num.DecimalFromFloat(0.05),
		CommitmentMinTimeFraction:   num.DecimalFromFloat(0.95),
		SlaCompetitionFactor:        num.DecimalFromFloat(0.9),
		PerformanceHysteresisEpochs: 1,
	}
}

func NewSnapshotEngine(
	config Config,
	log *logging.Logger,
	timeService TimeService,
	broker Broker,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	orderBook OrderBook,
	auctionState AuctionState,
	asset string,
	marketID string,
	stateVarEngine StateVarEngine,
	positionFactor num.Decimal,
	slaParams *types.LiquiditySLAParams,
	epochTime EpochTime,
) *SnapshotEngine {
	if slaParams == nil {
		slaParams = defaultLiquiditySLAParams()
	}

	e := NewEngine(
		config,
		log,
		timeService,
		broker,
		riskModel,
		priceMonitor,
		orderBook,
		auctionState,
		asset,
		marketID,
		stateVarEngine,
		positionFactor,
		slaParams,
	)

	se := &SnapshotEngine{
		Engine: e,
		snapshotV2: &snapshotV2{
			Engine:  e,
			pl:      types.Payload{},
			market:  marketID,
			stopped: false,
		},
		snapshotV1: &snapshotV1{},
	}

	se.buildHashKeys(marketID)

	return se
}
