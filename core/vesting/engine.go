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

package vesting

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/vesting ActivityStreakVestingMultiplier,Assets,Parties,StakeAccounting,Time

type Collateral interface {
	TransferVestedRewards(ctx context.Context, transfers []*types.Transfer) ([]*types.LedgerMovement, error)
	GetVestingRecovery() map[string]map[string]*num.Uint
	GetAllVestingQuantumBalance(party string) num.Decimal
	GetAllVestingAndVestedAccountForAsset(asset string) []*types.Account
}

type ActivityStreakVestingMultiplier interface {
	GetRewardsVestingMultiplier(party string) num.Decimal
}

type Broker interface {
	Send(events events.Event)
	Stage(event events.Event)
}

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

type Parties interface {
	RelatedKeys(key string) (*types.PartyID, []string)
}

type StakeAccounting interface {
	AddEvent(ctx context.Context, evt *types.StakeLinking)
}

type Time interface {
	GetTimeNow() time.Time
}

type PartyRewards struct {
	// the amounts per assets still being locked in the
	// account and not available to be released
	// this is a map of:
	// asset -> (remainingEpochLock -> Amount)
	Locked map[string]map[uint64]*num.Uint
	// the current part of the vesting account
	// per asset available for vesting
	Vesting map[string]*num.Uint
}

type MultiplierAndQuantBalance struct {
	Multiplier     num.Decimal
	QuantumBalance num.Decimal
}

type Engine struct {
	log *logging.Logger

	c      Collateral
	asvm   ActivityStreakVestingMultiplier
	broker Broker
	assets Assets

	minTransfer  num.Decimal
	baseRate     num.Decimal
	benefitTiers []*types.VestingBenefitTier

	state                map[string]*PartyRewards
	epochSeq             uint64
	upgradeHackActivated bool

	parties Parties

	// cache the reward bonus multiplier and quantum balance
	rewardBonusMultiplierCache map[string]MultiplierAndQuantBalance

	stakingAsset    string
	stakeAccounting StakeAccounting

	t Time
}

func New(
	log *logging.Logger,
	c Collateral,
	asvm ActivityStreakVestingMultiplier,
	broker Broker,
	assets Assets,
	parties Parties,
	t Time,
	stakeAccounting StakeAccounting,
) *Engine {
	log = log.Named(namedLogger)

	return &Engine{
		log:                        log,
		c:                          c,
		asvm:                       asvm,
		broker:                     broker,
		assets:                     assets,
		parties:                    parties,
		state:                      map[string]*PartyRewards{},
		rewardBonusMultiplierCache: map[string]MultiplierAndQuantBalance{},
		t:                          t,
		stakeAccounting:            stakeAccounting,
	}
}

func (e *Engine) OnCheckpointLoaded() {
	vestingBalances := e.c.GetVestingRecovery()
	for party, assetBalances := range vestingBalances {
		for asset, balance := range assetBalances {
			e.increaseVestingBalance(party, asset, balance.Clone())
		}
	}
}

func (e *Engine) OnBenefitTiersUpdate(_ context.Context, v interface{}) error {
	tiers, err := types.VestingBenefitTiersFromUntypedProto(v)
	if err != nil {
		return err
	}

	e.benefitTiers = tiers.Clone().Tiers
	sort.Slice(e.benefitTiers, func(i, j int) bool {
		return e.benefitTiers[i].MinimumQuantumBalance.LT(e.benefitTiers[j].MinimumQuantumBalance)
	})
	return nil
}

func (e *Engine) OnStakingAssetUpdate(_ context.Context, stakingAsset string) error {
	e.stakingAsset = stakingAsset
	return nil
}

func (e *Engine) OnRewardVestingBaseRateUpdate(_ context.Context, baseRate num.Decimal) error {
	e.baseRate = baseRate
	return nil
}

func (e *Engine) OnRewardVestingMinimumTransferUpdate(_ context.Context, minimumTransfer num.Decimal) error {
	e.minTransfer = minimumTransfer
	return nil
}

func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_END {
		e.clearMultiplierCache()
		e.moveLocked()
		e.distributeVested(ctx)
		e.broadcastVestingStatsUpdate(ctx, epoch.Seq)
		e.broadcastSummary(ctx, epoch.Seq)
		e.clearState()
		e.clearMultiplierCache()
	}
}

func (e *Engine) OnEpochRestore(_ context.Context, epoch types.Epoch) {
	e.epochSeq = epoch.Seq
}

func (e *Engine) updateStakingAccount(
	ctx context.Context,
	party string,
	amount *num.Uint,
	logIndex uint64,
	brokerFunc func(events.Event),
) {
	var (
		now       = e.t.GetTimeNow().Unix()
		height, _ = vgcontext.BlockHeightFromContext(ctx)
		txhash, _ = vgcontext.TxHashFromContext(ctx)
		id        = crypto.HashStrToHex(fmt.Sprintf("%v%v%v", party, txhash, height))
	)

	stakeLinking := &types.StakeLinking{
		ID:              id,
		Type:            types.StakeLinkingTypeDeposited,
		TS:              now,
		Party:           party,
		Amount:          amount,
		Status:          types.StakeLinkingStatusAccepted,
		FinalizedAt:     now,
		TxHash:          txhash,
		BlockHeight:     height,
		BlockTime:       now,
		LogIndex:        logIndex,
		EthereumAddress: party,
	}

	e.stakeAccounting.AddEvent(context.Background(), stakeLinking)
	brokerFunc(events.NewStakeLinking(ctx, *stakeLinking))
}

func (e *Engine) AddReward(
	ctx context.Context,
	party, asset string,
	amount *num.Uint,
	lockedForEpochs uint64,
) {
	// send to staking
	if asset == e.stakingAsset {
		e.updateStakingAccount(ctx, party, amount.Clone(), 1, e.broker.Send)
	}

	// no locktime, just increase the amount in vesting
	if lockedForEpochs == 0 {
		e.increaseVestingBalance(party, asset, amount)
		return
	}

	e.increaseLockedForAsset(party, asset, amount, lockedForEpochs)
}

func (e *Engine) rewardBonusMultiplier(quantumBalance num.Decimal) num.Decimal {
	multiplier := num.DecimalOne()

	for _, b := range e.benefitTiers {
		if quantumBalance.LessThan(num.DecimalFromUint(b.MinimumQuantumBalance)) {
			break
		}

		multiplier = b.RewardMultiplier
	}

	return multiplier
}

// GetSingleAndSummedRewardBonusMultipliers returns a single and summed reward bonus multipliers and quantum balances for a party.
// The single multiplier is calculated based on the quantum balance of the party.
// The summed multiplier is calculated based on the quantum balance of the party and all derived keys.
// Caches the summed multiplier and quantum balance for the party.
func (e *Engine) GetSingleAndSummedRewardBonusMultipliers(party string) (MultiplierAndQuantBalance, MultiplierAndQuantBalance) {
	owner := party

	partyID, derivedKeys := e.parties.RelatedKeys(party)
	if partyID != nil {
		owner = partyID.String()
	}

	ownerKey := fmt.Sprintf("owner-%s", owner)

	summed, foundSummed := e.rewardBonusMultiplierCache[ownerKey]

	for _, key := range append(derivedKeys, owner) {
		single, foundSingle := e.rewardBonusMultiplierCache[key]
		if !foundSingle {
			quantumBalanceForKey := e.c.GetAllVestingQuantumBalance(key)

			single.QuantumBalance = quantumBalanceForKey
			single.Multiplier = e.rewardBonusMultiplier(quantumBalanceForKey)
			e.rewardBonusMultiplierCache[key] = single
		}

		if !foundSummed {
			summed.QuantumBalance = summed.QuantumBalance.Add(single.QuantumBalance)
		}
	}

	if !foundSummed {
		summed.Multiplier = e.rewardBonusMultiplier(summed.QuantumBalance)
		e.rewardBonusMultiplierCache[ownerKey] = summed
	}

	return e.rewardBonusMultiplierCache[party], e.rewardBonusMultiplierCache[ownerKey]
}

func (e *Engine) getPartyRewards(party string) *PartyRewards {
	partyRewards, ok := e.state[party]
	if !ok {
		e.state[party] = &PartyRewards{
			Locked:  map[string]map[uint64]*num.Uint{},
			Vesting: map[string]*num.Uint{},
		}
		partyRewards = e.state[party]
	}

	return partyRewards
}

func (e *Engine) increaseLockedForAsset(
	party, asset string,
	amount *num.Uint,
	lockedForEpochs uint64,
) {
	partyRewards := e.getPartyRewards(party)
	locked, ok := partyRewards.Locked[asset]
	if !ok {
		locked = map[uint64]*num.Uint{}
	}
	amountLockedForEpochs, ok := locked[lockedForEpochs]
	if !ok {
		amountLockedForEpochs = num.UintZero()
	}
	amountLockedForEpochs.Add(amountLockedForEpochs, amount)
	locked[lockedForEpochs] = amountLockedForEpochs
	partyRewards.Locked[asset] = locked
}

func (e *Engine) increaseVestingBalance(
	party, asset string,
	amount *num.Uint,
) {
	partyRewards := e.getPartyRewards(party)

	vesting, ok := partyRewards.Vesting[asset]
	if !ok {
		vesting = num.UintZero()
	}
	vesting.Add(vesting, amount)
	partyRewards.Vesting[asset] = vesting
}

// moveLocked will move around locked funds.
// if the lock for epoch reach 0, the full amount
// is added to the vesting amount for the asset.
func (e *Engine) moveLocked() {
	for party, partyReward := range e.state {
		for asset, assetLocks := range partyReward.Locked {
			newLocked := map[uint64]*num.Uint{}
			for epochLeft, amount := range assetLocks {
				if epochLeft == 0 {
					e.increaseVestingBalance(party, asset, amount)
					continue
				}
				epochLeft--
				// just add the new map
				newLocked[epochLeft] = amount
			}

			// clear up if no rewards left
			if len(newLocked) <= 0 {
				delete(partyReward.Locked, asset)
				continue
			}

			partyReward.Locked[asset] = newLocked
		}
	}
}

func (e *Engine) distributeVested(ctx context.Context) {
	transfers := []*types.Transfer{}
	parties := maps.Keys(e.state)
	sort.Strings(parties)
	for _, party := range parties {
		rewards := e.state[party]
		assets := maps.Keys(rewards.Vesting)
		sort.Strings(assets)
		for _, asset := range assets {
			balance := rewards.Vesting[asset]
			transfer := e.makeTransfer(party, asset, balance.Clone())

			// we are clearing the account,
			// we can delete it.
			if transfer.MinAmount.EQ(balance) {
				delete(rewards.Vesting, asset)
			} else {
				rewards.Vesting[asset] = balance.Sub(balance, transfer.MinAmount)
			}

			transfers = append(transfers, transfer)
		}
	}

	// nothing to be done
	if len(transfers) == 0 {
		return
	}

	responses, err := e.c.TransferVestedRewards(ctx, transfers)
	if err != nil {
		e.log.Panic("could not transfer funds", logging.Error(err))
	}

	e.broker.Send(events.NewLedgerMovements(ctx, responses))
}

// OnTick is called on the beginning of the block. In here
// this is a post upgrade.
func (e *Engine) OnTick(ctx context.Context, _ time.Time) {
	if e.upgradeHackActivated {
		e.broadcastSummary(ctx, e.epochSeq)
		e.upgradeHackActivated = false
	}
}

func (e *Engine) makeTransfer(
	party, assetID string,
	balance *num.Uint,
) *types.Transfer {
	asset, _ := e.assets.Get(assetID)
	quantum := asset.Type().Details.Quantum
	minTransferAmount, _ := num.UintFromDecimal(quantum.Mul(e.minTransfer))

	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Asset: assetID,
		},
		Type: types.TransferTypeRewardsVested,
	}

	expectTransfer, _ := num.UintFromDecimal(
		balance.ToDecimal().Mul(e.baseRate).Mul(e.asvm.GetRewardsVestingMultiplier(party)),
	)

	// now we see which is the largest between the minimumTransfer
	// and the expected transfer
	expectTransfer = num.Max(expectTransfer, minTransferAmount)

	// and now we prevent any transfer to exceed the current balance
	expectTransfer = num.Min(expectTransfer, balance)

	transfer.Amount.Amount = expectTransfer.Clone()
	transfer.MinAmount = expectTransfer

	return transfer
}

func (e *Engine) clearState() {
	for party, v := range e.state {
		if len(v.Locked) == 0 && len(v.Vesting) == 0 {
			delete(e.state, party)
		}
	}
}

func (e *Engine) clearMultiplierCache() {
	e.rewardBonusMultiplierCache = map[string]MultiplierAndQuantBalance{}
}

func (e *Engine) broadcastSummary(ctx context.Context, seq uint64) {
	evt := &eventspb.VestingBalancesSummary{
		EpochSeq:              seq,
		PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
	}

	for p, pRewards := range e.state {
		if len(pRewards.Vesting) == 0 && len(pRewards.Locked) == 0 {
			continue
		}

		pSummary := &eventspb.PartyVestingSummary{
			Party:                p,
			PartyLockedBalances:  []*eventspb.PartyLockedBalance{},
			PartyVestingBalances: []*eventspb.PartyVestingBalance{},
		}

		// doing vesting first
		for asset, balance := range pRewards.Vesting {
			pSummary.PartyVestingBalances = append(
				pSummary.PartyVestingBalances,
				&eventspb.PartyVestingBalance{
					Asset:   asset,
					Balance: balance.String(),
				},
			)
		}

		sort.Slice(pSummary.PartyVestingBalances, func(i, j int) bool {
			return pSummary.PartyVestingBalances[i].Asset < pSummary.PartyVestingBalances[j].Asset
		})

		for asset, remainingEpochLockBalance := range pRewards.Locked {
			for remainingEpochs, balance := range remainingEpochLockBalance {
				pSummary.PartyLockedBalances = append(
					pSummary.PartyLockedBalances,
					&eventspb.PartyLockedBalance{
						Asset:      asset,
						Balance:    balance.String(),
						UntilEpoch: seq + remainingEpochs + 1, // we add one here because the remainingEpochs can be 0, meaning the funds are released next epoch
					},
				)
			}
		}

		sort.Slice(pSummary.PartyLockedBalances, func(i, j int) bool {
			if pSummary.PartyLockedBalances[i].Asset == pSummary.PartyLockedBalances[j].Asset {
				return pSummary.PartyLockedBalances[i].UntilEpoch < pSummary.PartyLockedBalances[j].UntilEpoch
			}
			return pSummary.PartyLockedBalances[i].Asset < pSummary.PartyLockedBalances[j].Asset
		})

		evt.PartiesVestingSummary = append(evt.PartiesVestingSummary, pSummary)
	}

	sort.Slice(evt.PartiesVestingSummary, func(i, j int) bool {
		return evt.PartiesVestingSummary[i].Party < evt.PartiesVestingSummary[j].Party
	})

	e.broker.Send(events.NewVestingBalancesSummaryEvent(ctx, evt))
}

func (e *Engine) broadcastVestingStatsUpdate(ctx context.Context, seq uint64) {
	evt := &eventspb.VestingStatsUpdated{
		AtEpoch: seq,
		Stats:   make([]*eventspb.PartyVestingStats, 0, len(e.state)),
	}

	parties := maps.Keys(e.state)
	slices.Sort(parties)

	for _, party := range parties {
		single, summed := e.GetSingleAndSummedRewardBonusMultipliers(party)
		// To avoid excessively large decimals.
		single.QuantumBalance.Round(2)
		summed.QuantumBalance.Round(2)
		evt.Stats = append(evt.Stats, &eventspb.PartyVestingStats{
			PartyId:                     party,
			RewardBonusMultiplier:       single.Multiplier.String(),
			QuantumBalance:              single.QuantumBalance.String(),
			SummedRewardBonusMultiplier: summed.Multiplier.String(),
			SummedQuantumBalance:        summed.QuantumBalance.String(),
		})
	}

	e.broker.Send(events.NewVestingStatsUpdatedEvent(ctx, evt))
}
