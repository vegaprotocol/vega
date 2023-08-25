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
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var (
	ErrIsAlreadyAReferee = func(party types.PartyID) error {
		return fmt.Errorf("party %q has already been referred", party)
	}

	ErrIsAlreadyAReferrer = func(party types.PartyID) error {
		return fmt.Errorf("party %q is already a referrer", party)
	}

	ErrUnknownReferralCode = func(code types.ReferralSetID) error {
		return fmt.Errorf("no referral set for referral code %q", code)
	}
)

type Engine struct {
	broker                Broker
	marketActivityTracker MarketActivityTracker
	timeSvc               TimeService

	currentEpoch uint64

	// referralSetsNotionalVolumes tracks the notional volumes per teams. Each
	// element of the num.Uint array is an epoch.
	referralSetsNotionalVolumes *runningVolumes

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

	sets      map[types.ReferralSetID]*types.ReferralSet
	referrers map[types.PartyID]types.ReferralSetID
	referees  map[types.PartyID]types.ReferralSetID
}

func (e *Engine) SetExists(setID types.ReferralSetID) bool {
	_, ok := e.sets[setID]
	return ok
}

func (e *Engine) CreateReferralSet(ctx context.Context, party types.PartyID, deterministicSetID types.ReferralSetID) error {
	if _, ok := e.referrers[party]; ok {
		return ErrIsAlreadyAReferrer(party)
	}
	if _, ok := e.referees[party]; ok {
		return ErrIsAlreadyAReferee(party)
	}

	now := e.timeSvc.GetTimeNow()

	newSet := types.ReferralSet{
		ID:        deterministicSetID,
		CreatedAt: now,
		UpdatedAt: now,
		Referrer: &types.Membership{
			PartyID:        party,
			JoinedAt:       now,
			StartedAtEpoch: e.currentEpoch,
		},
	}

	e.sets[deterministicSetID] = &newSet
	e.referrers[party] = deterministicSetID

	e.broker.Send(events.NewReferralSetCreatedEvent(ctx, &newSet))

	return nil
}

func (e *Engine) ApplyReferralCode(ctx context.Context, party types.PartyID, setID types.ReferralSetID) error {
	if _, ok := e.referees[party]; ok {
		return ErrIsAlreadyAReferee(party)
	}

	if _, ok := e.referrers[party]; ok {
		return ErrIsAlreadyAReferrer(party)
	}

	set, ok := e.sets[setID]
	if !ok {
		return ErrUnknownReferralCode(setID)
	}

	now := e.timeSvc.GetTimeNow()

	set.UpdatedAt = now

	membership := &types.Membership{
		PartyID:        party,
		JoinedAt:       now,
		StartedAtEpoch: e.currentEpoch,
	}
	set.Referees = append(set.Referees, membership)

	e.referees[party] = set.ID

	e.broker.Send(events.NewRefereeJoinedReferralSetEvent(ctx, setID, membership))

	return nil
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

	setID, isReferrer := e.referrers[party]
	if !isReferrer {
		// This party is not eligible to referral program rewards.
		return num.DecimalZero()
	}

	runningTeamVolume := e.referralSetsNotionalVolumes.RunningSetVolumeForWindow(setID, e.currentProgram.WindowLength)

	tiersLen := len(e.currentProgram.BenefitTiers)

	for i := tiersLen - 1; i >= 0; i-- {
		tier := e.currentProgram.BenefitTiers[i]
		if runningTeamVolume.GTE(tier.MinimumRunningNotionalTakerVolume) {
			return tier.ReferralRewardFactor
		}
	}

	return num.DecimalZero()
}

func (e *Engine) DiscountFactorForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	setID, isReferee := e.referees[party]
	if !isReferee {
		// This party is not eligible to referral program discount.
		return num.DecimalZero()
	}

	epochCount := uint64(0)
	set := e.sets[setID]
	for _, referee := range set.Referees {
		if referee.PartyID == party {
			epochCount = e.currentEpoch - referee.StartedAtEpoch
			break
		}
	}

	runningTeamVolume := e.referralSetsNotionalVolumes.RunningSetVolumeForWindow(setID, e.currentProgram.WindowLength)

	tiersLen := len(e.currentProgram.BenefitTiers)

	for i := tiersLen - 1; i >= 0; i-- {
		tier := e.currentProgram.BenefitTiers[i]
		if epochCount >= tier.MinimumEpochs.Uint64() && runningTeamVolume.GTE(tier.MinimumRunningNotionalTakerVolume) {
			return tier.ReferralDiscountFactor
		}
	}

	return num.DecimalZero()
}

func (e *Engine) OnReferralProgramMaxPartyNotionalVolumeByQuantumPerEpochUpdate(_ context.Context, value *num.Uint) error {
	e.referralSetsNotionalVolumes.maxPartyNotionalVolumeByQuantumPerEpoch = value
	return nil
}

func (e *Engine) OnEpoch(ctx context.Context, ep types.Epoch) {
	switch ep.Action {
	case vegapb.EpochAction_EPOCH_ACTION_START:
		e.currentEpoch = ep.Seq
		e.applyProgramUpdate(ctx, ep.StartTime)
	case vegapb.EpochAction_EPOCH_ACTION_END:
		e.computeReferralSetsNotionalRunningVolume(ep)
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, ep types.Epoch) {
	if ep.Action == vegapb.EpochAction_EPOCH_ACTION_START {
		e.currentEpoch = ep.Seq
	}
}

func (e *Engine) applyProgramUpdate(ctx context.Context, startEpochTime time.Time) {
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
	// time.
	if e.currentProgram != nil && !e.currentProgram.EndOfProgramTimestamp.After(startEpochTime) {
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

func (e *Engine) loadReferralSetsFromSnapshot(sets *snapshotpb.ReferralSets) {
	if sets == nil {
		return
	}

	for _, set := range sets.Sets {
		setID := types.ReferralSetID(set.Id)

		newSet := &types.ReferralSet{
			ID:        setID,
			CreatedAt: time.Unix(0, set.CreatedAt),
			UpdatedAt: time.Unix(0, set.CreatedAt),
			Referrer: &types.Membership{
				PartyID:        types.PartyID(set.Referrer.PartyId),
				JoinedAt:       time.Unix(0, set.Referrer.JoinedAt),
				StartedAtEpoch: set.Referrer.StartedAtEpoch,
			},
		}

		e.referrers[types.PartyID(set.Referrer.PartyId)] = setID

		for _, r := range set.Referees {
			partyID := types.PartyID(r.PartyId)
			e.referees[partyID] = setID
			newSet.Referees = append(newSet.Referees,
				&types.Membership{
					PartyID:        partyID,
					JoinedAt:       time.Unix(0, r.JoinedAt),
					StartedAtEpoch: r.StartedAtEpoch,
				},
			)
		}

		e.sets[setID] = newSet
	}
}

func (e *Engine) computeReferralSetsNotionalRunningVolume(epoch types.Epoch) {
	for partyID, setID := range e.referrers {
		volumeForEpoch := e.marketActivityTracker.NotionalTakerVolumeForParty(string(partyID))
		e.referralSetsNotionalVolumes.Add(epoch.Seq, setID, volumeForEpoch)
	}

	for partyID, setID := range e.referees {
		volumeForEpoch := e.marketActivityTracker.NotionalTakerVolumeForParty(string(partyID))
		e.referralSetsNotionalVolumes.Add(epoch.Seq, setID, volumeForEpoch)
	}
}

func NewEngine(epochEngine EpochEngine, broker Broker, timeSvc TimeService, mat MarketActivityTracker) *Engine {
	engine := &Engine{
		broker:                broker,
		timeSvc:               timeSvc,
		marketActivityTracker: mat,

		// There is no program yet, so we mark it has ended so consumer of this
		// engine can know there is no reward computation to be done.
		programHasEnded: true,

		referralSetsNotionalVolumes: newRunningVolumes(),

		sets:      map[types.ReferralSetID]*types.ReferralSet{},
		referrers: map[types.PartyID]types.ReferralSetID{},
		referees:  map[types.PartyID]types.ReferralSetID{},
	}

	epochEngine.NotifyOnEpoch(engine.OnEpoch, engine.OnEpochRestore)

	return engine
}
