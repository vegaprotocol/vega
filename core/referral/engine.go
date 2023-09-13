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
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const MaximumWindowLength uint64 = 100

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

	ErrNotEligibleForReferralRewards = func(party string, balance, required *num.Uint) error {
		return fmt.Errorf("party %q not eligible for referral rewards, staking balance required of %s got %s", party, required.String(), balance.String())
	}

	ErrNotPartOfAReferralSet = func(party types.PartyID) error {
		return fmt.Errorf("party %q is not part of a referral set", party)
	}

	ErrUnknownSetID = errors.New("unknown set ID")
)

type Engine struct {
	broker                Broker
	marketActivityTracker MarketActivityTracker
	timeSvc               TimeService

	currentEpoch uint64
	staking      StakingBalances

	// referralSetsNotionalVolumes tracks the notional volumes per teams. Each
	// element of the num.Uint array is an epoch.
	referralSetsNotionalVolumes *runningVolumes
	factorsByReferee            map[types.PartyID]*types.RefereeStats

	// referralProgramMinStakedVegaTokens is the minimum number of token a party
	// must possess to become and stay a referrer.
	referralProgramMinStakedVegaTokens *num.Uint

	rewardProportionUpdate num.Decimal

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

func (e *Engine) GetReferrer(referee types.PartyID) (types.PartyID, error) {
	setID, ok := e.referees[referee]
	if !ok {
		return "", ErrNotPartOfAReferralSet(referee)
	}

	return e.sets[setID].Referrer.PartyID, nil
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

	if err := e.isPartyEligible(string(party)); err != nil {
		return err
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
	if _, ok := e.referrers[party]; ok {
		return ErrIsAlreadyAReferrer(party)
	}

	var (
		isSwitching bool
		prevSet     types.ReferralSetID
		ok          bool
	)
	if prevSet, ok = e.referees[party]; ok {
		isSwitching = e.canSwitchReferralSet(party, setID)
		if !isSwitching {
			return ErrIsAlreadyAReferee(party)
		}
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

	if isSwitching {
		e.removeFromSet(party, prevSet)
	}

	return nil
}

func (e *Engine) removeFromSet(party types.PartyID, prevSet types.ReferralSetID) {
	set := e.sets[prevSet]

	var idx int
	for i, r := range set.Referees {
		if r.PartyID == party {
			idx = i
			break
		}
	}

	set.Referees = append(set.Referees[:idx], set.Referees[idx+1:]...)
}

func (e *Engine) UpdateProgram(newProgram *types.ReferralProgram) {
	e.latestProgramVersion += 1
	e.newProgram = newProgram

	sort.Slice(e.newProgram.BenefitTiers, func(i, j int) bool {
		return e.newProgram.BenefitTiers[i].MinimumRunningNotionalTakerVolume.LT(e.newProgram.BenefitTiers[j].MinimumRunningNotionalTakerVolume)
	})

	sort.Slice(e.newProgram.StakingTiers, func(i, j int) bool {
		return e.newProgram.StakingTiers[i].MinimumStakedTokens.LT(e.newProgram.StakingTiers[j].MinimumStakedTokens)
	})

	e.newProgram.Version = e.latestProgramVersion
}

func (e *Engine) HasProgramEnded() bool {
	return e.programHasEnded
}

func (e *Engine) ReferralDiscountFactorForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	factors, ok := e.factorsByReferee[party]
	if !ok {
		return num.DecimalZero()
	}

	return factors.DiscountFactor
}

func (e *Engine) RewardsFactorForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	factors, ok := e.factorsByReferee[party]
	if !ok {
		return num.DecimalZero()
	}

	return factors.RewardFactor
}

func (e *Engine) RewardsFactorMultiplierAppliedForParty(party types.PartyID) num.Decimal {
	return num.MinD(
		e.RewardsFactorForParty(party).Mul(e.RewardsMultiplierForParty(party)),
		e.rewardProportionUpdate,
	)
}

func (e *Engine) RewardsMultiplierForParty(party types.PartyID) num.Decimal {
	if e.programHasEnded {
		return num.DecimalZero()
	}

	setID, isReferee := e.referees[party]
	if !isReferee {
		// This party is not eligible to referral program rewards.
		return num.DecimalZero()
	}

	if e.isSetEligible(setID) != nil {
		return num.DecimalZero()
	}

	balance, _ := e.staking.GetAvailableBalance(
		string(e.sets[setID].Referrer.PartyID),
	)

	multiplier := num.DecimalOne()
	for _, v := range e.currentProgram.StakingTiers {
		if balance.LTE(v.MinimumStakedTokens) {
			break
		}
		multiplier = v.ReferralRewardMultiplier
	}

	return multiplier
}

func (e *Engine) VolumeDiscountFactorForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (e *Engine) OnReferralProgramMaxReferralRewardProportionUpdate(_ context.Context, value num.Decimal) error {
	e.rewardProportionUpdate = value
	return nil
}

func (e *Engine) OnReferralProgramMinStakedVegaTokensUpdate(_ context.Context, value *num.Uint) error {
	e.referralProgramMinStakedVegaTokens = value
	return nil
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
		e.computeReferralSetsStats(ctx, ep)
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

func (e *Engine) notifyReferralSetStatsUpdated(ctx context.Context, stats *types.ReferralSetStats) {
	e.broker.Send(events.NewReferralSetStatsUpdatedEvent(ctx, stats))
}

func (e *Engine) loadCurrentReferralProgramFromSnapshot(program *vegapb.ReferralProgram) {
	if program == nil {
		e.currentProgram = nil
		return
	}

	e.currentProgram = types.NewReferralProgramFromProto(program)
	e.programHasEnded = false

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

func (e *Engine) loadReferralSetsFromSnapshot(setsProto *snapshotpb.ReferralSets) {
	if setsProto == nil {
		return
	}

	for _, setProto := range setsProto.Sets {
		setID := types.ReferralSetID(setProto.Id)

		newSet := &types.ReferralSet{
			ID:        setID,
			CreatedAt: time.Unix(0, setProto.CreatedAt),
			UpdatedAt: time.Unix(0, setProto.CreatedAt),
			Referrer: &types.Membership{
				PartyID:        types.PartyID(setProto.Referrer.PartyId),
				JoinedAt:       time.Unix(0, setProto.Referrer.JoinedAt),
				StartedAtEpoch: setProto.Referrer.StartedAtEpoch,
			},
		}

		e.referrers[types.PartyID(setProto.Referrer.PartyId)] = setID

		for _, r := range setProto.Referees {
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

		runningVolumes := make([]*notionalVolume, 0, len(setProto.RunningVolumes))
		for _, volume := range setProto.RunningVolumes {
			var volumeNum *num.Uint
			if len(volume.Volume) > 0 {
				volumeNum = num.UintFromBytes(volume.Volume)
			}
			runningVolumes = append(runningVolumes, &notionalVolume{
				epoch: volume.Epoch,
				value: volumeNum,
			})
		}

		// set only if the running volume is not empty, or it will panic
		// down the line when trying to add new ones.
		// the creation of runningVolumeBySet is done in the Add method of the
		// runningVolumes type.
		if len(runningVolumes) > 0 {
			e.referralSetsNotionalVolumes.runningVolumesBySet[setID] = runningVolumes
		}

		e.sets[setID] = newSet
	}
}

func (e *Engine) computeReferralSetsStats(ctx context.Context, epoch types.Epoch) {
	priorEpoch := uint64(0)
	if epoch.Seq > MaximumWindowLength {
		priorEpoch = epoch.Seq - MaximumWindowLength
	}
	e.referralSetsNotionalVolumes.RemovePriorEpoch(priorEpoch)

	for partyID, setID := range e.referrers {
		volumeForEpoch := e.marketActivityTracker.NotionalTakerVolumeForParty(string(partyID))
		e.referralSetsNotionalVolumes.Add(epoch.Seq, setID, volumeForEpoch)
	}

	for partyID, setID := range e.referees {
		volumeForEpoch := e.marketActivityTracker.NotionalTakerVolumeForParty(string(partyID))
		e.referralSetsNotionalVolumes.Add(epoch.Seq, setID, volumeForEpoch)
	}

	if e.programHasEnded {
		return
	}

	e.computeFactorsByReferee(ctx, epoch.Seq)
}

func (e *Engine) computeFactorsByReferee(ctx context.Context, epoch uint64) {
	e.factorsByReferee = map[types.PartyID]*types.RefereeStats{}

	allStats := map[types.ReferralSetID]*types.ReferralSetStats{}

	for setID := range e.sets {
		allStats[setID] = &types.ReferralSetStats{
			AtEpoch:                  epoch,
			SetID:                    setID,
			RefereesStats:            map[types.PartyID]*types.RefereeStats{},
			ReferralSetRunningVolume: e.referralSetsNotionalVolumes.RunningSetVolumeForWindow(setID, e.currentProgram.WindowLength),
		}
	}

	tiersLen := len(e.currentProgram.BenefitTiers)

	for party, setID := range e.referees {
		set := e.sets[setID]
		epochCount := uint64(0)

		for _, referee := range set.Referees {
			if referee.PartyID == party {
				epochCount = e.currentEpoch - referee.StartedAtEpoch + 1
				break
			}
		}

		setStats := allStats[setID]
		runningVolumeForSet := setStats.ReferralSetRunningVolume

		refereeStats := &types.RefereeStats{}
		e.factorsByReferee[party] = refereeStats
		setStats.RefereesStats[party] = refereeStats

		for i := tiersLen - 1; i >= 0; i-- {
			tier := e.currentProgram.BenefitTiers[i]
			if refereeStats.DiscountFactor.Equal(num.DecimalZero()) && epochCount >= tier.MinimumEpochs.Uint64() && runningVolumeForSet.GTE(tier.MinimumRunningNotionalTakerVolume) {
				refereeStats.DiscountFactor = tier.ReferralDiscountFactor
			}
			if refereeStats.RewardFactor.Equal(num.DecimalZero()) && runningVolumeForSet.GTE(tier.MinimumRunningNotionalTakerVolume) {
				refereeStats.RewardFactor = tier.ReferralRewardFactor
			}
		}
	}

	setIDs := maps.Keys(allStats)
	slices.Sort(setIDs)
	for _, setID := range setIDs {
		e.notifyReferralSetStatsUpdated(ctx, allStats[setID])
	}
}

func (e *Engine) isSetEligible(setID types.ReferralSetID) error {
	set, ok := e.sets[setID]
	if !ok {
		return ErrUnknownSetID
	}

	return e.isPartyEligible(string(set.Referrer.PartyID))
}

func (e *Engine) canSwitchReferralSet(party types.PartyID, newSet types.ReferralSetID) bool {
	currentSet := e.referees[party]
	if currentSet == newSet {
		return false
	}

	// if the current set is not eligible for rewards,
	// then we can switch
	if e.isSetEligible(currentSet) != nil {
		return true
	}

	return false
}

func (e *Engine) isPartyEligible(party string) error {
	// Ignore error, function returns zero balance anyway.
	balance, _ := e.staking.GetAvailableBalance(party)

	if balance.GTE(e.referralProgramMinStakedVegaTokens) {
		return nil
	}

	return ErrNotEligibleForReferralRewards(party, balance, e.referralProgramMinStakedVegaTokens)
}

func NewEngine(broker Broker, timeSvc TimeService, mat MarketActivityTracker, staking StakingBalances) *Engine {
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
		staking:   staking,
	}

	return engine
}
