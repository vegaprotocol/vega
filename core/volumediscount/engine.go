// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package volumediscount

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

const MaximumWindowLength uint64 = 100

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
	}
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case vegapb.EpochAction_EPOCH_ACTION_START:
		e.applyProgramUpdate(ctx, ep.StartTime)
		if e.currentProgram != nil {
			e.calculatePartiesVolumeForWindow(int(e.currentProgram.WindowLength))
		}
	case vegapb.EpochAction_EPOCH_ACTION_END:
		e.updateNotionalVolumeForEpoch()
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

func (e *Engine) VolumeDiscountFactorForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	notionalVolume := e.avgVolumePerParty[party]
	tiersLen := len(e.currentProgram.VolumeBenefitTiers)
	for i := tiersLen - 1; i >= 0; i-- {
		tier := e.currentProgram.VolumeBenefitTiers[i]
		if notionalVolume.GreaterThanOrEqual(tier.MinimumRunningNotionalTakerVolume.ToDecimal()) {
			return tier.VolumeDiscountFactor
		}
	}

	return num.DecimalZero()
}

func (e *Engine) applyProgramUpdate(ctx context.Context, startEpochTime time.Time) {
	if e.newProgram != nil {
		if e.currentProgram != nil {
			e.endCurrentProgram()
			e.startNewProgram()
			e.notifyVolumeDiscountProgramUpdated(ctx)
		} else {
			e.startNewProgram()
			e.notifyVolumeDiscountProgramStarted(ctx)
		}
	}

	// This handles a edge case where the new program ends before the next
	// epoch starts. It can happen when the proposal updating the volume discount
	// program doesn't specify an end date that is to close to the enactment
	// time.
	if e.currentProgram != nil && !e.currentProgram.EndOfProgramTimestamp.After(startEpochTime) {
		e.notifyVolumeDiscountProgramEnded(ctx)
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

func (e *Engine) notifyVolumeDiscountProgramStarted(ctx context.Context) {
	e.broker.Send(events.NewVolumeDiscountProgramStartedEvent(ctx, e.currentProgram))
}

func (e *Engine) notifyVolumeDiscountProgramUpdated(ctx context.Context) {
	e.broker.Send(events.NewVolumeDiscountProgramUpdatedEvent(ctx, e.currentProgram))
}

func (e *Engine) notifyVolumeDiscountProgramEnded(ctx context.Context) {
	e.broker.Send(events.NewVolumeDiscountProgramEndedEvent(ctx, e.currentProgram.Version, e.currentProgram.ID))
}

func (e *Engine) calculatePartiesVolumeForWindow(windowSize int) {
	windowSizeAsDecimal := num.DecimalFromInt64(int64(windowSize))
	for pi := range e.parties {
		total := num.UintZero()
		for i := 0; i < windowSize; i++ {
			valueForEpoch, ok := e.epochData[(e.epochDataIndex+int(MaximumWindowLength)-i-1)%int(MaximumWindowLength)][pi]
			if !ok {
				valueForEpoch = num.UintZero()
			}
			total.AddSum(valueForEpoch)
		}
		e.avgVolumePerParty[pi] = total.ToDecimal().Div(windowSizeAsDecimal)
	}
}

func (e *Engine) updateNotionalVolumeForEpoch() {
	e.epochData[e.epochDataIndex] = e.marketActivityTracker.NotionalTakerVolumeForAllParties()
	for pi := range e.epochData[e.epochDataIndex] {
		e.parties[pi] = struct{}{}
	}
	e.epochDataIndex = (e.epochDataIndex + 1) % int(MaximumWindowLength)
}
