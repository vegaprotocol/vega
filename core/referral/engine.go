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
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var (
	ErrIsAlreadyAReferee = func(party string) error {
		return fmt.Errorf("party %v has already been referred", party)
	}
	// FIXME: This is not possible
	ErrIsAlreadyAReferrer = func(party string) error {
		return fmt.Errorf("party %v is already a referrer", party)
	}
)

type TimeService interface {
	GetTimeNow() time.Time
}

type Engine struct {
	broker      Broker
	teamsEngine TeamsEngine

	timeSvc TimeService

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

	// TODO: snapshot this
	// keep track of all referral sets
	// referral set ID -> Referral set
	sets map[string]*types.ReferralSet

	// NO need for snapshot, dynamically computed
	// map of referrer to set ID
	referrers map[string]string

	// NO need for snapshot, dynamically computed
	// map of referees to set ID
	referees map[string]string
}

func (e *Engine) SetExists(setID string) bool {
	_, ok := e.sets[setID]
	return ok
}

func (e *Engine) CreateReferralSet(ctx context.Context, party string, set *commandspb.CreateReferralSet, deterministicID string) error {
	if _, ok := e.referrers[party]; ok {
		return ErrIsAlreadyAReferrer(party)
	}

	now := e.timeSvc.GetTimeNow()

	newSet := types.ReferralSet{
		ID:        deterministicID,
		CreatedAt: now,
		UpdatedAt: now,
		Referrer: &types.Membership{
			PartyID:       types.PartyID(party),
			JoinedAt:      now,
			NumberOfEpoch: 0,
		},
	}

	e.sets[deterministicID] = &newSet

	e.referrers[party] = deterministicID

	e.broker.Send(events.NewReferralSetCreatedEvent(ctx, &newSet))

	return nil
}

func (e *Engine) ApplyReferralCode(ctx context.Context, party string, cset *commandspb.ApplyReferralCode) error {
	if _, ok := e.referees[party]; ok {
		return ErrIsAlreadyAReferee(party)
	}

	if _, ok := e.referrers[party]; ok {
		return ErrIsAlreadyAReferrer(party)
	}

	set, ok := e.sets[cset.Id]
	if !ok {
		return fmt.Errorf("invalid referral code %v", cset.Id)
	}

	now := e.timeSvc.GetTimeNow()

	set.UpdatedAt = now
	set.Referees = append(set.Referees, &types.Membership{
		PartyID:       types.PartyID(party),
		JoinedAt:      now,
		NumberOfEpoch: 0,
	})

	e.broker.Send(events.NewRefereeJoinedReferralSetEvent(ctx, cset.Id, party, now))

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

func (e *Engine) loadReferralSetsFromSnapshot(sets *snapshotpb.ReferralSets) {
	if sets == nil {
		return
	}

	for _, set := range sets.Sets {
		newSet := &types.ReferralSet{
			ID:        set.Id,
			CreatedAt: time.Unix(0, set.CreatedAt),
			UpdatedAt: time.Unix(0, set.CreatedAt),
			Referrer: &types.Membership{
				PartyID:       types.PartyID(set.Referrer.PartyId),
				JoinedAt:      time.Unix(0, set.Referrer.JoinedAt),
				NumberOfEpoch: set.Referrer.NumberOfEpoch,
			},
		}

		e.referrers[set.Referrer.PartyId] = set.Id

		for _, r := range set.Referrees {
			e.referees[r.PartyId] = set.Id
			newSet.Referees = append(newSet.Referees,
				&types.Membership{
					PartyID:       types.PartyID(r.PartyId),
					JoinedAt:      time.Unix(0, r.JoinedAt),
					NumberOfEpoch: r.NumberOfEpoch,
				},
			)
		}

		e.sets[set.Id] = newSet
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

// TODO: inject time service
func NewEngine(epochEngine EpochEngine, broker Broker, teamsEngine TeamsEngine) *Engine {
	engine := &Engine{
		broker:      broker,
		teamsEngine: teamsEngine,
		// There is no program yet, so we mark it has ended so consumer of this
		// engine can know there is no reward computation to be done.
		programHasEnded:      true,
		latestProgramVersion: 0,

		sets:      map[string]*types.ReferralSet{},
		referrers: map[string]string{},
		referees:  map[string]string{},
	}

	epochEngine.NotifyOnEpoch(engine.OnEpoch, engine.OnEpochRestore)

	return engine
}
