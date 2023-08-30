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

package vesting

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"golang.org/x/exp/maps"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/vesting Collateral,ActivityStreakVestingMultiplier,Broker,Assets

type Collateral interface {
	TransferVestedRewards(
		ctx context.Context, transfers []*types.Transfer,
	) ([]*types.LedgerMovement, error)
	GetVestingRecovery() map[string]map[string]*num.Uint
	GetAllVestingQuantumBalance(party string) *num.Uint
}

type ActivityStreakVestingMultiplier interface {
	Get(party string) num.Decimal
}

type Broker interface {
	Send(events events.Event)
}

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
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

type Engine struct {
	log *logging.Logger

	c      Collateral
	asvm   ActivityStreakVestingMultiplier
	broker Broker
	assets Assets

	minTransfer  num.Decimal
	baseRate     num.Decimal
	benefitTiers []*types.VestingBenefitTier

	state map[string]*PartyRewards
}

func New(
	log *logging.Logger,
	c Collateral,
	asvm ActivityStreakVestingMultiplier,
	broker Broker,
	assets Assets,
) *Engine {
	log = log.Named(namedLogger)

	return &Engine{
		log:    log,
		c:      c,
		asvm:   asvm,
		broker: broker,
		assets: assets,
		state:  map[string]*PartyRewards{},
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

func (e *Engine) OnBenefitTiersUpdate(
	_ context.Context, v interface{},
) error {
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

func (e *Engine) OnRewardVestingBaseRateUpdate(
	_ context.Context, baseRate num.Decimal,
) error {
	e.baseRate = baseRate
	return nil
}

func (e *Engine) OnRewardVestingMinimumTransferUpdate(
	_ context.Context, minimumTransfer num.Decimal,
) error {
	e.minTransfer = minimumTransfer
	return nil
}

func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_END {
		e.moveLocked()
		e.distributeVested(ctx)
		e.clearup()
	}
}

func (e *Engine) OnEpochRestore(ctx context.Context, epoch types.Epoch) {}

func (e *Engine) AddReward(
	party, asset string,
	amount *num.Uint,
	lockedForEpochs uint64,
) {
	// no locktime, just increase the amount in vesting
	if lockedForEpochs == 0 {
		e.increaseVestingBalance(
			party, asset, amount,
		)
		return
	}

	e.increaseLockedForAsset(
		party, asset, amount, lockedForEpochs,
	)
}

func (e *Engine) GetRewardsBonusMultiplier(party string) num.Decimal {
	quantumBalance := e.c.GetAllVestingQuantumBalance(party)

	multiplier := num.DecimalOne()

	for _, b := range e.benefitTiers {
		if quantumBalance.LT(b.MinimumQuantumBalance) {
			break
		}

		multiplier = b.RewardMultiplier
	}

	return multiplier
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

// checkLocked will move around locked funds.
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
	for _, party := range parties {
		rewards := e.state[party]
		assets := maps.Keys(rewards.Vesting)
		for _, asset := range assets {
			balance := rewards.Vesting[asset]
			transfer := e.makeTransfer(
				party, asset, balance.Clone(),
			)

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
	if len(transfers) <= 0 {
		return
	}

	responses, err := e.c.TransferVestedRewards(ctx, transfers)
	if err != nil {
		e.log.Panic("could not transfer funds", logging.Error(err))
	}

	e.broker.Send(events.NewLedgerMovements(ctx, responses))
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
		balance.ToDecimal().Mul(e.baseRate).Mul(e.asvm.Get(party)),
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

// just remove party entries once they are not needed anymore.
func (e *Engine) clearup() {
	for party, v := range e.state {
		if len(v.Locked) <= 0 && len(v.Vesting) <= 0 {
			delete(e.state, party)
		}
	}
}

// TODO implement me. if there's no multiplier return 1.
func (e *Engine) GetRewardBonusMultiplier(party string) num.Decimal {
	return num.DecimalOne()
}
