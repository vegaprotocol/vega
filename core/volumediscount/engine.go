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

package volumediscount

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

	epochData            []map[types.PartyID]*num.Uint
	epochDataIndex       int
	parties              map[types.PartyID]struct{}
	avgVolumePerParty    map[types.PartyID]num.Decimal
	latestProgramVersion uint64
	currentProgram       *types.VolumeDiscountProgram
	newProgram           *types.VolumeDiscountProgram
	programHasEnded      bool

	factorsByParty map[types.PartyID]types.VolumeDiscountStats
}

func New(broker Broker, marketActivityTracker MarketActivityTracker) *Engine {
	return &Engine{
		broker:                broker,
		marketActivityTracker: marketActivityTracker,
		epochData:             make([]map[types.PartyID]*num.Uint, MaximumWindowLength),
		epochDataIndex:        0,
		avgVolumePerParty:     map[types.PartyID]num.Decimal{},
		parties:               map[types.PartyID]struct{}{},
		programHasEnded:       true,
		factorsByParty:        map[types.PartyID]types.VolumeDiscountStats{},
	}
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case vegapb.EpochAction_EPOCH_ACTION_START:
		// whatever current program is
		pp := e.currentProgram
		e.applyProgramUpdate(ctx, ep.StartTime, ep.Seq)
		// we have an active program, and it's not the same one after we called applyProgramUpdate -> update factors.
		if !e.programHasEnded && pp != e.currentProgram {
			// calculate volume for the window of the new program
			e.calculatePartiesVolumeForWindow(int(e.currentProgram.WindowLength))
			// update the factors
			e.computeFactorsByParty(ctx, ep.Seq)
		}
	case vegapb.EpochAction_EPOCH_ACTION_END:
		e.updateNotionalVolumeForEpoch()
		if !e.programHasEnded {
			e.calculatePartiesVolumeForWindow(int(e.currentProgram.WindowLength))
			e.computeFactorsByParty(ctx, ep.Seq)
		}
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, ep types.Epoch) {
}

func (e *Engine) UpdateProgram(newProgram *types.VolumeDiscountProgram) {
	e.latestProgramVersion += 1
	e.newProgram = newProgram

	sort.Slice(e.newProgram.VolumeBenefitTiers, func(i, j int) bool {
		return e.newProgram.VolumeBenefitTiers[i].MinimumRunningNotionalTakerVolume.LT(e.newProgram.VolumeBenefitTiers[j].MinimumRunningNotionalTakerVolume)
	})

	e.newProgram.Version = e.latestProgramVersion
}

func (e *Engine) HasProgramEnded() bool {
	return e.programHasEnded
}

func (e *Engine) VolumeDiscountFactorForParty(party types.PartyID) types.Factors {
	if e.programHasEnded {
		return types.EmptyFactors
	}

	factors, ok := e.factorsByParty[party]
	if !ok {
		return types.EmptyFactors
	}

	return factors.DiscountFactors
}

func (e *Engine) TakerNotionalForParty(party types.PartyID) num.Decimal {
	return e.avgVolumePerParty[party]
}

func (e *Engine) applyProgramUpdate(ctx context.Context, startEpochTime time.Time, epoch uint64) {
	if e.newProgram != nil {
		if e.currentProgram != nil {
			e.endCurrentProgram()
			e.startNewProgram()
			e.notifyVolumeDiscountProgramUpdated(ctx, startEpochTime, epoch)
		} else {
			e.startNewProgram()
			e.notifyVolumeDiscountProgramStarted(ctx, startEpochTime, epoch)
		}
	}

	// This handles a edge case where the new program ends before the next
	// epoch starts. It can happen when the proposal updating the volume discount
	// program specifies an end date that is within the same epoch as the enactment
	// time.
	if e.currentProgram != nil && !e.currentProgram.EndOfProgramTimestamp.IsZero() && !e.currentProgram.EndOfProgramTimestamp.After(startEpochTime) {
		e.notifyVolumeDiscountProgramEnded(ctx, startEpochTime, epoch)
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

func (e *Engine) notifyVolumeDiscountProgramStarted(ctx context.Context, epochTime time.Time, epoch uint64) {
	e.broker.Send(events.NewVolumeDiscountProgramStartedEvent(ctx, e.currentProgram, epochTime, epoch))
}

func (e *Engine) notifyVolumeDiscountProgramUpdated(ctx context.Context, epochTime time.Time, epoch uint64) {
	e.broker.Send(events.NewVolumeDiscountProgramUpdatedEvent(ctx, e.currentProgram, epochTime, epoch))
}

func (e *Engine) notifyVolumeDiscountProgramEnded(ctx context.Context, epochTime time.Time, epoch uint64) {
	e.broker.Send(events.NewVolumeDiscountProgramEndedEvent(ctx, e.currentProgram.Version, e.currentProgram.ID, epochTime, epoch))
}

func (e *Engine) calculatePartiesVolumeForWindow(windowSize int) {
	for pi := range e.parties {
		total := num.UintZero()
		for i := 0; i < windowSize; i++ {
			valueForEpoch, ok := e.epochData[(e.epochDataIndex+int(MaximumWindowLength)-i-1)%int(MaximumWindowLength)][pi]
			if !ok {
				valueForEpoch = num.UintZero()
			}
			total.AddSum(valueForEpoch)
		}
		e.avgVolumePerParty[pi] = total.ToDecimal()
	}
}

func (e *Engine) updateNotionalVolumeForEpoch() {
	e.epochData[e.epochDataIndex] = e.marketActivityTracker.NotionalTakerVolumeForAllParties()
	for pi := range e.epochData[e.epochDataIndex] {
		e.parties[pi] = struct{}{}
	}
	e.epochDataIndex = (e.epochDataIndex + 1) % int(MaximumWindowLength)
}

func (e *Engine) computeFactorsByParty(ctx context.Context, epoch uint64) {
	e.factorsByParty = map[types.PartyID]types.VolumeDiscountStats{}

	parties := maps.Keys(e.avgVolumePerParty)
	slices.Sort(parties)

	tiersLen := len(e.currentProgram.VolumeBenefitTiers)

	evt := &eventspb.VolumeDiscountStatsUpdated{
		AtEpoch: epoch,
		Stats:   make([]*eventspb.PartyVolumeDiscountStats, 0, len(e.avgVolumePerParty)),
	}

	for _, party := range parties {
		notionalVolume := e.avgVolumePerParty[party]
		qualifiedForTier := false
		for i := tiersLen - 1; i >= 0; i-- {
			tier := e.currentProgram.VolumeBenefitTiers[i]
			if notionalVolume.GreaterThanOrEqual(tier.MinimumRunningNotionalTakerVolume.ToDecimal()) {
				e.factorsByParty[party] = types.VolumeDiscountStats{
					DiscountFactors: tier.VolumeDiscountFactors,
				}
				evt.Stats = append(evt.Stats, &eventspb.PartyVolumeDiscountStats{
					PartyId:         party.String(),
					DiscountFactors: tier.VolumeDiscountFactors.IntoDiscountFactorsProto(),
					RunningVolume:   notionalVolume.Round(0).String(),
				})
				qualifiedForTier = true
				break
			}
		}
		// if the party hasn't qualified, then still send the stats but with a zero factor
		if !qualifiedForTier {
			evt.Stats = append(evt.Stats, &eventspb.PartyVolumeDiscountStats{
				PartyId:         party.String(),
				DiscountFactors: types.EmptyFactors.IntoDiscountFactorsProto(),
				RunningVolume:   notionalVolume.Round(0).String(),
			})
		}
	}

	e.broker.Send(events.NewVolumeDiscountStatsUpdatedEvent(ctx, evt))
}
