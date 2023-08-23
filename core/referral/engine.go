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

package referral

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type Engine struct {
	broker      Broker
	teamsEngine TeamsEngine

	// maxPartyNotionalVolumeByQuantumPerEpoch limits the volume in quantum units
	// which is eligible each epoch for referral program mechanisms.
	maxPartyNotionalVolumeByQuantumPerEpoch *num.Uint

	// latestProgramVersion tracks the latest version of the program. It used to
	// value any new program that comes in. It starts at 1.
	// It's incremented every time an update is received. Therefore, if, during
	// the same epoch, we have 2 successive updates, this field will be incremented
	// twice.
	latestProgramVersion uint64

	// currentProgram is the program currently in used against which the reward
	// are computed.
	// It's `nil` is there is none.
	currentProgram *types.ReferralProgram

	// programHasEnded tells if the current program has reached it's
	// end. It's flipped at the end of the epoch.
	programHasEnded bool
	// newProgram is the program born from the last enacted UpdateReferralProgram
	// proposal to apply at the start of the next epoch.
	// It's `nil` is there is none.
	newProgram *types.ReferralProgram
}

func (e *Engine) UpdateProgram(newProgram *types.ReferralProgram) {
	e.latestProgramVersion += 1
	e.newProgram = newProgram
	e.newProgram.Version = e.latestProgramVersion
}

func (e *Engine) HasProgramEnded() bool {
	return e.programHasEnded
}

func (e *Engine) RewardsFactorForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	if !e.teamsEngine.IsTeamMember(party) {
		// This party is not eligible to referral program rewards.
		return num.DecimalZero()
	}

	epochCount := e.teamsEngine.NumberOfEpochInTeamForParty(party)

	tier := e.findTierByEpochCount(epochCount)
	if tier == nil {
		// This party has not stayed in a team long enough to match a tier.
		return num.DecimalZero()
	}

	return tier.ReferralRewardFactor
}

func (e *Engine) OnReferralProgramMaxPartyNotionalVolumeByQuantumPerEpochUpdate(_ context.Context, value *num.Uint) error {
	e.maxPartyNotionalVolumeByQuantumPerEpoch = value
	return nil
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case vegapb.EpochAction_EPOCH_ACTION_END:
		e.applyUpdate(ctx, ep.EndTime)
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, _ types.Epoch) {}

func (e *Engine) applyUpdate(ctx context.Context, epochEnd time.Time) {
	if e.newProgram != nil {
		if e.currentProgram != nil {
			e.endCurrentProgram()
			e.startNewProgram()
			e.notifyReferralProgramUpdated(ctx)
		} else {
			e.startNewProgram()
			e.notifyReferralProgramStarted(ctx)
		}
	}

	// This handles a edge case where the new program ends before the next
	// epoch starts. It can happen when the proposal updating the referral
	// program doesn't specify an end date that is to close to the enactment
	// time. That is believed to happen
	if e.currentProgram != nil && !e.currentProgram.EndOfProgramTimestamp.After(epochEnd) {
		e.notifyReferralProgramEnded(ctx)
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

func (e *Engine) notifyReferralProgramStarted(ctx context.Context) {
	e.broker.Send(events.NewReferralProgramStartedEvent(ctx, e.currentProgram))
}

func (e *Engine) notifyReferralProgramUpdated(ctx context.Context) {
	e.broker.Send(events.NewReferralProgramUpdatedEvent(ctx, e.currentProgram))
}

func (e *Engine) notifyReferralProgramEnded(ctx context.Context) {
	e.broker.Send(events.NewReferralProgramEndedEvent(ctx, e.currentProgram.Version, e.currentProgram.ID))
}

func (e *Engine) loadCurrentReferralProgramFromSnapshot(program *vegapb.ReferralProgram) {
	if program == nil {
		e.currentProgram = nil
		return
	}

	e.currentProgram = types.NewReferralProgramFromProto(program)

	if e.latestProgramVersion < e.currentProgram.Version {
		e.latestProgramVersion = e.currentProgram.Version
	}
}

func (e *Engine) loadNewReferralProgramFromSnapshot(program *vegapb.ReferralProgram) {
	if program == nil {
		e.newProgram = nil
		return
	}

	e.newProgram = types.NewReferralProgramFromProto(program)

	if e.latestProgramVersion < e.newProgram.Version {
		e.latestProgramVersion = e.newProgram.Version
	}
}

func (e *Engine) findTierByEpochCount(epochCount uint64) *types.BenefitTier {
	tiersLen := len(e.currentProgram.BenefitTiers)

	for i := tiersLen - 1; i >= 0; i-- {
		tier := e.currentProgram.BenefitTiers[i]
		if epochCount >= tier.MinimumEpochs.Uint64() {
			return tier
		}
	}

	return nil
}

func NewEngine(epochEngine EpochEngine, broker Broker, teamsEngine TeamsEngine) *Engine {
	engine := &Engine{
		broker:      broker,
		teamsEngine: teamsEngine,

		// There is no program yet, so we mark it has ended so consumer of this
		// engine can know there is no reward computation to be done.
		programHasEnded: true,

		latestProgramVersion: 0,
	}

	epochEngine.NotifyOnEpoch(engine.OnEpoch, engine.OnEpochRestore)

	return engine
}
