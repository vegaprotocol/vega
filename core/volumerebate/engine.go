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

package volumerebate

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const MaximumWindowLength uint64 = 200

type Engine struct {
	broker                Broker
	marketActivityTracker MarketActivityTracker

	parties                           map[string]struct{}
	fractionPerParty                  map[string]num.Decimal
	makerFeesReceivedInWindowPerParty map[string]*num.Uint
	latestProgramVersion              uint64
	currentProgram                    *types.VolumeRebateProgram
	newProgram                        *types.VolumeRebateProgram
	programHasEnded                   bool

	factorsByParty      map[types.PartyID]types.VolumeRebateStats
	buyBackFee          num.Decimal
	treasureFee         num.Decimal
	maxAdditionalRebate num.Decimal
}

func New(broker Broker, marketActivityTracker MarketActivityTracker) *Engine {
	return &Engine{
		broker:                            broker,
		marketActivityTracker:             marketActivityTracker,
		fractionPerParty:                  map[string]num.Decimal{},
		makerFeesReceivedInWindowPerParty: map[string]*num.Uint{},
		parties:                           map[string]struct{}{},
		programHasEnded:                   true,
		factorsByParty:                    map[types.PartyID]types.VolumeRebateStats{},
	}
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case vegapb.EpochAction_EPOCH_ACTION_START:
		e.applyProgramUpdate(ctx, ep.StartTime, ep.Seq)
	case vegapb.EpochAction_EPOCH_ACTION_END:
		e.updateState()
		if !e.programHasEnded {
			e.computeFactorsByParty(ctx, ep.Seq)
		}
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, ep types.Epoch) {
}

func (e *Engine) UpdateProgram(newProgram *types.VolumeRebateProgram) {
	e.latestProgramVersion += 1
	e.newProgram = newProgram

	sort.Slice(e.newProgram.VolumeRebateBenefitTiers, func(i, j int) bool {
		return e.newProgram.VolumeRebateBenefitTiers[i].MinimumPartyMakerVolumeFraction.LessThan(e.newProgram.VolumeRebateBenefitTiers[j].MinimumPartyMakerVolumeFraction)
	})

	e.newProgram.Version = e.latestProgramVersion
}

func (e *Engine) HasProgramEnded() bool {
	return e.programHasEnded
}

func (e *Engine) VolumeRebateFactorForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	factors, ok := e.factorsByParty[party]
	if !ok {
		return num.DecimalZero()
	}

	// this is needed here again because the factors are calculated at the end of the epoch and
	// the fee factors may change during the epoch so to ensure the factor is capped at any time
	// we apply the min again here
	return e.effectiveAdditionalRebate(factors.RebateFactor)
}

func (e *Engine) MakerVolumeFractionForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	frac, ok := e.fractionPerParty[party.String()]
	if !ok {
		return num.DecimalZero()
	}

	return frac
}

func (e *Engine) applyProgramUpdate(ctx context.Context, startEpochTime time.Time, epoch uint64) {
	if e.newProgram != nil {
		if e.currentProgram != nil {
			e.endCurrentProgram()
			e.startNewProgram()
			e.notifyVolumeRebateProgramUpdated(ctx, startEpochTime, epoch)
		} else {
			e.startNewProgram()
			e.notifyVolumeRebateProgramStarted(ctx, startEpochTime, epoch)
		}
	}

	// This handles a edge case where the new program ends before the next
	// epoch starts. It can happen when the proposal updating the volume discount
	// program specifies an end date that is within the same epoch as the enactment
	// time.
	if e.currentProgram != nil && !e.currentProgram.EndOfProgramTimestamp.IsZero() && !e.currentProgram.EndOfProgramTimestamp.After(startEpochTime) {
		e.notifyVolumeRebateProgramEnded(ctx, startEpochTime, epoch)
		e.endCurrentProgram()
	}
}

func (e *Engine) endCurrentProgram() {
	e.programHasEnded = true
	e.currentProgram = nil
}

func (e *Engine) startNewProgram() {
	e.programHasEnded = false
	e.currentProgram = e.newProgram
	e.newProgram = nil
}

func (e *Engine) notifyVolumeRebateProgramStarted(ctx context.Context, epochTime time.Time, epoch uint64) {
	e.broker.Send(events.NewVolumeRebateProgramStartedEvent(ctx, e.currentProgram, epochTime, epoch))
}

func (e *Engine) notifyVolumeRebateProgramUpdated(ctx context.Context, epochTime time.Time, epoch uint64) {
	e.broker.Send(events.NewVolumeRebateProgramUpdatedEvent(ctx, e.currentProgram, epochTime, epoch))
}

func (e *Engine) notifyVolumeRebateProgramEnded(ctx context.Context, epochTime time.Time, epoch uint64) {
	e.broker.Send(events.NewVolumeRebateProgramEndedEvent(ctx, e.currentProgram.Version, e.currentProgram.ID, epochTime, epoch))
}

func (e *Engine) updateState() {
	if e.currentProgram == nil {
		return
	}
	e.makerFeesReceivedInWindowPerParty, e.fractionPerParty = e.marketActivityTracker.CalculateTotalMakerContributionInQuantum(int(e.currentProgram.WindowLength))
	for p := range e.fractionPerParty {
		if _, ok := e.factorsByParty[types.PartyID(p)]; !ok {
			e.factorsByParty[types.PartyID(p)] = types.VolumeRebateStats{
				RebateFactor: num.DecimalZero(),
			}
		}
	}
}

func (e *Engine) computeFactorsByParty(ctx context.Context, epoch uint64) {
	parties := maps.Keys(e.factorsByParty)
	slices.Sort(parties)

	e.factorsByParty = map[types.PartyID]types.VolumeRebateStats{}

	tiersLen := len(e.currentProgram.VolumeRebateBenefitTiers)

	evt := &eventspb.VolumeRebateStatsUpdated{
		AtEpoch: epoch,
		Stats:   make([]*eventspb.PartyVolumeRebateStats, 0, len(parties)),
	}

	for _, party := range parties {
		makerFraction := e.fractionPerParty[party.String()]
		receivedFees, ok := e.makerFeesReceivedInWindowPerParty[party.String()]
		if !ok {
			receivedFees = num.UintZero()
		}
		qualifiedForTier := false
		for i := tiersLen - 1; i >= 0; i-- {
			tier := e.currentProgram.VolumeRebateBenefitTiers[i]
			if makerFraction.GreaterThanOrEqual(tier.MinimumPartyMakerVolumeFraction) {
				e.factorsByParty[party] = types.VolumeRebateStats{
					RebateFactor: e.effectiveAdditionalRebate(tier.AdditionalMakerRebate),
				}
				evt.Stats = append(evt.Stats, &eventspb.PartyVolumeRebateStats{
					PartyId:             party.String(),
					AdditionalRebate:    tier.AdditionalMakerRebate.String(),
					MakerVolumeFraction: makerFraction.String(),
					MakerFeesReceived:   receivedFees.String(),
				})
				qualifiedForTier = true
				break
			}
		}
		// if the party hasn't qualified, then still send the stats but with a zero factor
		if !qualifiedForTier {
			evt.Stats = append(evt.Stats, &eventspb.PartyVolumeRebateStats{
				PartyId:             party.String(),
				AdditionalRebate:    "0",
				MakerVolumeFraction: makerFraction.String(),
				MakerFeesReceived:   receivedFees.String(),
			})
		}
	}

	e.broker.Send(events.NewVolumeRebateStatsUpdatedEvent(ctx, evt))
}

func (e *Engine) OnMarketFeeFactorsTreasuryFeeUpdate(ctx context.Context, d num.Decimal) error {
	e.treasureFee = d
	e.maxAdditionalRebate = e.treasureFee.Add(e.buyBackFee)
	return nil
}

func (e *Engine) OnMarketFeeFactorsBuyBackFeeUpdate(ctx context.Context, d num.Decimal) error {
	e.buyBackFee = d
	e.maxAdditionalRebate = e.treasureFee.Add(e.buyBackFee)
	return nil
}

func (e *Engine) effectiveAdditionalRebate(tierRebate num.Decimal) num.Decimal {
	return num.MinD(e.maxAdditionalRebate, tierRebate)
}
