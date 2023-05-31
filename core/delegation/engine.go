// Copyright (c) 2022 Gobalsky Labs Limited
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

package delegation

import (
	"context"
	"encoding/hex"
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var minRatioForAutoDelegation, _ = num.DecimalFromString("0.95")

const reconciliationInterval = 30 * time.Second

var (
	activeKey    = (&types.PayloadDelegationActive{}).Key()
	pendingKey   = (&types.PayloadDelegationPending{}).Key()
	autoKey      = (&types.PayloadDelegationAuto{}).Key()
	lastReconKey = (&types.PayloadDelegationLastReconTime{}).Key()
)

var (
	// ErrPartyHasNoStakingAccount is returned when the staking account for the party cannot be found.
	ErrPartyHasNoStakingAccount = errors.New("cannot find staking account for the party")
	// ErrInvalidNodeID is returned when the node id passed for delegation/undelegation is not a validator node identifier.
	ErrInvalidNodeID = errors.New("invalid node ID")
	// ErrInsufficientBalanceForDelegation is returned when the balance in the staking account is insufficient to cover all committed and pending delegations.
	ErrInsufficientBalanceForDelegation = errors.New("insufficient balance for delegation")
	// ErrIncorrectTokenAmountForUndelegation is returned when the amount to undelegation doesn't match the delegation balance (pending + committed) for the party and validator.
	ErrIncorrectTokenAmountForUndelegation = errors.New("incorrect token amount for undelegation")
	// ErrAmountLTMinAmountForDelegation is returned when the amount to delegate to a node is lower than the minimum allowed amount from network params.
	ErrAmountLTMinAmountForDelegation = errors.New("delegation amount is lower than the minimum amount for delegation for a validator")
)

// TimeService notifies the reward engine on time updates
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/rewards TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// ValidatorTopology represents the topology of validators and can check if a given node is a validator.
type ValidatorTopology interface {
	IsValidatorNodeID(nodeID string) bool
	AllNodeIDs() []string
	Get(key string) *validators.ValidatorData
}

// Broker send events
// we no longer need to generate this mock here, we can use the broker/mocks package instead.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// StakingAccounts provides access to the staking balance of a given party now and within a duration of an epoch.
type StakingAccounts interface {
	GetAvailableBalance(party string) (*num.Uint, error)
	GetAvailableBalanceInRange(party string, from, to time.Time) (*num.Uint, error)
}

type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
}

// party delegation state - how much is delegated by the party to each validator and in total.
type partyDelegation struct {
	party          string               // party ID
	nodeToAmount   map[string]*num.Uint // nodeID -> delegated amount
	totalDelegated *num.Uint            // total amount delegated by party
}

// Engine is handling the delegations balances from parties to validators
// The delegation engine is designed in the following way with the following assumptions:
// 1. during epoch it is called with delegation requests that update the delegation balance of the party for the next epoch
// 2. At the end of the epoch:
// 2.1 updates the delegated balances to reconcile the epoch's staking account balance for each party such that if a party withdrew from their
//
//	staking account during the epoch it will not count for them for rewarding
//
// 2.2 capture the state after 2.1 to be returned to the rewarding engine
// 2.3 process all pending delegations.
type Engine struct {
	log                      *logging.Logger
	config                   Config
	broker                   Broker
	topology                 ValidatorTopology           // an interface to the topoology to interact with validator nodes if needed
	stakingAccounts          StakingAccounts             // an interface to the staking account for getting party balances
	partyDelegationState     map[string]*partyDelegation // party to active delegation balances
	nextPartyDelegationState map[string]*partyDelegation // party to next epoch delegation balances
	minDelegationAmount      *num.Uint                   // min delegation amount per delegation request
	currentEpoch             types.Epoch                 // the current epoch for pending delegations
	autoDelegationMode       map[string]struct{}         // parties entered auto-delegation mode
	dss                      *delegationSnapshotState    // snapshot state
	lastReconciliation       time.Time                   // last time staking balance has been reconciled against delegation balance
}

// New instantiates a new delegation engine.
func New(log *logging.Logger, config Config, broker Broker, topology ValidatorTopology, stakingAccounts StakingAccounts, epochEngine EpochEngine, ts TimeService) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:                   config,
		log:                      log,
		broker:                   broker,
		topology:                 topology,
		stakingAccounts:          stakingAccounts,
		partyDelegationState:     map[string]*partyDelegation{},
		nextPartyDelegationState: map[string]*partyDelegation{},
		autoDelegationMode:       map[string]struct{}{},
		dss:                      &delegationSnapshotState{},
		lastReconciliation:       time.Time{},
	}
	// register for epoch notifications
	epochEngine.NotifyOnEpoch(e.onEpochEvent, e.onEpochRestore)

	return e
}

func (e *Engine) Hash() []byte {
	buf, err := e.Checkpoint()
	if err != nil {
		e.log.Panic("could not create checkpoint", logging.Error(err))
	}
	h := crypto.Hash(buf)
	e.log.Debug("delegations state hash", logging.String("hash", hex.EncodeToString(h)))
	return h
}

// OnMinAmountChanged updates the network parameter for minDelegationAmount.
func (e *Engine) OnMinAmountChanged(ctx context.Context, minAmount num.Decimal) error {
	e.minDelegationAmount, _ = num.UintFromDecimal(minAmount)
	return nil
}

// every few blocks try to reconcile the association and nomination for the current and next epoch.
func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	// if we've already done reconciliation (i.e. not first epoch) and it's been over <reconciliationIntervalSeconds> since, then reconcile.
	if (e.lastReconciliation != time.Time{}) && t.Sub(e.lastReconciliation) >= reconciliationInterval {
		// always reconcile the balance from the start of the epoch to the current time for simplicity
		e.reconcileAssociationWithNomination(ctx, e.currentEpoch.StartTime, t, e.currentEpoch.Seq)
	}
}

// update the current epoch at which current pending delegations are recorded
// regardless if the event is start or stop of the epoch. the sequence is what identifies the epoch.
func (e *Engine) onEpochEvent(ctx context.Context, epoch types.Epoch) {
	if (e.lastReconciliation == time.Time{}) {
		e.lastReconciliation = epoch.StartTime
	}
	if epoch.Seq != e.currentEpoch.Seq {
		// emit an event for the next epoch's delegations
		for _, p := range e.sortParties(e.nextPartyDelegationState) {
			for _, n := range e.sortNodes(e.nextPartyDelegationState[p].nodeToAmount) {
				e.sendDelegatedBalanceEvent(ctx, p, n, epoch.Seq+1, e.nextPartyDelegationState[p].nodeToAmount[n])
			}
		}
	}
	e.currentEpoch = epoch
}

// reconcileAssociationWithNomination adjusts if necessary the nomination balance with the association balance for the current and next epoch.
func (e *Engine) reconcileAssociationWithNomination(ctx context.Context, from, to time.Time, epochSeq uint64) {
	// for current epoch we reconcile against the minimum balance for the epoch as given by the partial function
	e.reconcile(ctx, e.partyDelegationState, e.stakeInRangeFunc(from, to), epochSeq)
	// for the next epoch we reconcile against the current balance
	e.reconcile(ctx, e.nextPartyDelegationState, e.stakingAccounts.GetAvailableBalance, epochSeq+1)
	e.lastReconciliation = to
}

// reconcile checks if there is a mismatch between the amount associated with VEGA by a party and the amount nominated by this party. If a mismatch is found it is auto-adjusted.
func (e *Engine) reconcile(ctx context.Context, delegationState map[string]*partyDelegation, stakeFunc func(string) (*num.Uint, error), epochSeq uint64) {
	parties := e.sortParties(delegationState)
	for _, party := range parties {
		stakeBalance, err := stakeFunc(party)
		if err != nil {
			e.log.Error("Failed to get available balance", logging.Error(err))
			continue
		}

		// if the stake covers the total delegated balance nothing to do further for the party
		if stakeBalance.GTE(delegationState[party].totalDelegated) {
			continue
		}

		partyDelegation := delegationState[party]
		// if the stake account balance for the epoch is less than the delegated balance - we need to undelegate the difference
		// this will be done evenly as much as possible between all validators with delegation from the party
		remainingBalanceToUndelegate := num.UintZero().Sub(partyDelegation.totalDelegated, stakeBalance)
		totalTaken := num.UintZero()
		nodeIDs := e.sortNodes(partyDelegation.nodeToAmount)

		// undelegate proportionally across delegated validator nodes
		totalDeletation := partyDelegation.totalDelegated.Clone()
		for _, nodeID := range nodeIDs {
			balance := partyDelegation.nodeToAmount[nodeID]
			balanceToTake := num.UintZero().Mul(balance, remainingBalanceToUndelegate)
			balanceToTake = num.UintZero().Div(balanceToTake, totalDeletation)

			if balanceToTake.IsZero() {
				continue
			}

			e.decreaseBalanceAndFireEvent(ctx, party, nodeID, balanceToTake, epochSeq, delegationState, false, false)
			totalTaken = num.Sum(totalTaken, balanceToTake)
		}

		// if there was a remainder, the maximum that we need to take more from each node is 1,
		if totalTaken.LT(remainingBalanceToUndelegate) {
			for _, nodeID := range nodeIDs {
				balance, ok := partyDelegation.nodeToAmount[nodeID]
				if !ok {
					continue
				}
				if totalTaken.EQ(remainingBalanceToUndelegate) {
					break
				}
				if !balance.IsZero() {
					e.decreaseBalanceAndFireEvent(ctx, party, nodeID, num.NewUint(1), epochSeq, delegationState, false, false)
					totalTaken = num.Sum(totalTaken, num.NewUint(1))
				}
			}
		}

		currentNodeIDs := e.sortNodes(delegationState[party].nodeToAmount)
		for _, nodeID := range currentNodeIDs {
			e.sendDelegatedBalanceEvent(ctx, party, nodeID, epochSeq, delegationState[party].nodeToAmount[nodeID])
			if amt, ok := delegationState[party].nodeToAmount[nodeID]; ok {
				if amt.IsZero() {
					delete(delegationState[party].nodeToAmount, nodeID)
				}
			}
		}

		if state, ok := delegationState[party]; ok {
			if state.totalDelegated.IsZero() {
				delete(delegationState, party)
			}
		}

		// get out of auto delegation mode
		delete(e.autoDelegationMode, party)
	}
}

// Delegate updates the delegation balance for the next epoch.
func (e *Engine) Delegate(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	amt := amount.Clone()

	// check if the node is a validator node
	if !e.topology.IsValidatorNodeID(nodeID) {
		e.log.Error("Trying to delegate to an invalid node", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID))
		return ErrInvalidNodeID
	}

	// check if the delegator has a staking account
	partyBalance, err := e.stakingAccounts.GetAvailableBalance(party)
	if err != nil {
		e.log.Error("Party has no staking account balance", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID))
		return ErrPartyHasNoStakingAccount
	}

	// check if the amount for delegation is valid
	if amt.LT(e.minDelegationAmount) {
		e.log.Error("Amount for delegation is lower than minimum required amount", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID), logging.String("amount", num.UintToString(amount)), logging.String("minAmount", num.UintToString(e.minDelegationAmount)))
		return ErrAmountLTMinAmountForDelegation
	}

	// get the pending balance for the next epoch
	nextEpochBalance := num.UintZero()
	if nextEpoch, ok := e.nextPartyDelegationState[party]; ok {
		nextEpochBalance = nextEpoch.totalDelegated
	}

	// if the projected balance for next epoch is greater than the current staking account balance reject the transaction
	if num.Sum(nextEpochBalance, amt).GT(partyBalance) {
		e.log.Error("Party has insufficient account balance", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID), logging.String("associatedBalance", num.UintToString(partyBalance)), logging.String("delegationBalance", num.UintToString(nextEpochBalance)), logging.String("amount", num.UintToString(amount)))
		return ErrInsufficientBalanceForDelegation
	}

	// update the balance for next epoch
	if _, ok := e.nextPartyDelegationState[party]; !ok {
		e.nextPartyDelegationState[party] = &partyDelegation{
			party:          party,
			totalDelegated: num.UintZero(),
			nodeToAmount:   map[string]*num.Uint{nodeID: num.UintZero()},
		}
	}

	// update next epoch's balance and send an event
	nextEpochState := e.nextPartyDelegationState[party]
	nextEpochState.totalDelegated.AddSum(amt)
	if _, ok := nextEpochState.nodeToAmount[nodeID]; !ok {
		nextEpochState.nodeToAmount[nodeID] = num.UintZero()
	}
	nextEpochState.nodeToAmount[nodeID].AddSum(amt)
	e.sendDelegatedBalanceEvent(ctx, party, nodeID, e.currentEpoch.Seq+1, e.nextPartyDelegationState[party].nodeToAmount[nodeID])
	return nil
}

// UndelegateAtEndOfEpoch increases the pending undelegation balance and potentially decreases the pending delegation balance for a given validator node and party.
func (e *Engine) UndelegateAtEndOfEpoch(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	// check if the node is a validator node
	if e.topology == nil || !e.topology.IsValidatorNodeID(nodeID) {
		e.log.Error("Trying to delegate to an invalid node", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID))
		return ErrInvalidNodeID
	}

	// get the balance for next epoch
	nextEpochBalanceOnNode := num.UintZero()
	if nextEpoch, ok := e.nextPartyDelegationState[party]; ok {
		if nodeAmount, ok := nextEpoch.nodeToAmount[nodeID]; ok {
			nextEpochBalanceOnNode = nodeAmount
		}
	}

	// if the request is for undelegating the whole balance set the amount to the total balance
	amt := amount.Clone()
	if amt.IsZero() {
		amt = nextEpochBalanceOnNode.Clone()
	}

	// if the amount is greater than the available balance to undelegate return error
	if amt.GT(nextEpochBalanceOnNode) {
		e.log.Error("Invalid undelegation - trying to undelegate more than delegated", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID), logging.String("undelegationAmount", num.UintToString(amt)), logging.String("totalDelegationBalance", num.UintToString(nextEpochBalanceOnNode)))
		return ErrIncorrectTokenAmountForUndelegation
	}

	// update next epoch's balance and send an event
	e.decreaseBalanceAndFireEvent(ctx, party, nodeID, amt, e.currentEpoch.Seq+1, e.nextPartyDelegationState, true, true)

	// get out of auto delegation mode as the party made explicit undelegations
	delete(e.autoDelegationMode, party)
	return nil
}

// UndelegateNow changes the balance of delegation immediately without waiting for the end of the epoch
// if possible it removed balance from pending delegated, if not enough it removes balance from the current epoch delegated amount.
func (e *Engine) UndelegateNow(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	// check if the node is a validator node
	if e.topology == nil || !e.topology.IsValidatorNodeID(nodeID) {
		e.log.Error("Trying to delegate to an invalid node", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID))
		return ErrInvalidNodeID
	}

	// the purpose of this is that if a party has x delegated in the current epoch and x + a delegated for the next epoch, undelegateNow will start with undelegating
	// the current epoch but if there's any left it will undelegate from the next epoch. This is unlikely to happen but still
	currentEpochBalanceOnNode := num.UintZero()
	if epoch, ok := e.partyDelegationState[party]; ok {
		if nodeAmount, ok := epoch.nodeToAmount[nodeID]; ok {
			currentEpochBalanceOnNode = nodeAmount
		}
	}
	nextEpochBalanceOnNode := num.UintZero()
	if epoch, ok := e.nextPartyDelegationState[party]; ok {
		if nodeAmount, ok := epoch.nodeToAmount[nodeID]; ok {
			nextEpochBalanceOnNode = nodeAmount
		}
	}

	epochBalanceOnNode := num.Max(currentEpochBalanceOnNode, nextEpochBalanceOnNode)

	// if the request is for undelegating the whole balance set the amount to the total balance
	amt := amount.Clone()
	if amt.IsZero() {
		amt = epochBalanceOnNode.Clone()
	}

	// if the amount is greater than the available balance to undelegate return error
	if amt.GT(epochBalanceOnNode) {
		e.log.Error("Invalid undelegation - trying to undelegate more than delegated", logging.Uint64("epoch", e.currentEpoch.Seq), logging.String("party", party), logging.String("validator", nodeID), logging.String("undelegationAmount", num.UintToString(amt)), logging.String("totalDelegationBalance", num.UintToString(epochBalanceOnNode)))
		return ErrIncorrectTokenAmountForUndelegation
	}

	undelegateFromCurrentEpoch := num.Min(currentEpochBalanceOnNode, amt)
	if !undelegateFromCurrentEpoch.IsZero() {
		e.decreaseBalanceAndFireEvent(ctx, party, nodeID, undelegateFromCurrentEpoch, e.currentEpoch.Seq, e.partyDelegationState, true, true)
	}

	undelegateFromNextEpoch := num.Min(nextEpochBalanceOnNode, amt)
	if !undelegateFromNextEpoch.IsZero() {
		e.decreaseBalanceAndFireEvent(ctx, party, nodeID, undelegateFromNextEpoch, e.currentEpoch.Seq+1, e.nextPartyDelegationState, true, true)
	}

	// get out of auto delegation mode
	delete(e.autoDelegationMode, party)
	return nil
}

// ProcessEpochDelegations updates the delegation engine state at the end of a given epoch and returns the validation-delegation data for rewarding for that epoch
// step 1: process delegation data for the epoch - undelegate if the balance of the staking account doesn't cover all delegations
// step 2: capture validator delegation data to be returned
// step 3: apply pending undelegations
// step 4: apply pending delegations
// step 5: apply auto delegations
// epoch here is the epoch that ended.
func (e *Engine) ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData {
	if e.log.IsDebug() {
		e.log.Debug("ProcessEpochDelegations:", logging.Time("start", epoch.StartTime), logging.Time("end", epoch.EndTime))
	}

	// check balance for the epoch duration and undelegate if delegations don't have sufficient cover
	// the state of the engine by the end of this method reflects the state to be used for reward engine.
	e.reconcileAssociationWithNomination(ctx, epoch.StartTime, epoch.EndTime, epoch.Seq)
	stateForRewards := e.getValidatorData()

	// promote pending delegations

	excludeFromAutoDelegation := map[string]struct{}{}
	for p, state := range e.nextPartyDelegationState {
		for n, nAmt := range state.nodeToAmount {
			if currState, ok := e.partyDelegationState[p]; ok {
				if currAmt, ok := currState.nodeToAmount[n]; ok {
					if currAmt.NEQ(nAmt) {
						excludeFromAutoDelegation[p] = struct{}{}
					}
				} else {
					excludeFromAutoDelegation[p] = struct{}{}
				}
			} else {
				excludeFromAutoDelegation[p] = struct{}{}
			}
		}
	}

	next := e.prepareNextEpochDelegationState()
	e.partyDelegationState = e.nextPartyDelegationState
	e.nextPartyDelegationState = next

	// process auto delegations
	// this is updating the state for the epoch that's about to begin therefore it needs to have incremented sequence
	e.processAutoDelegation(ctx, e.eligiblePartiesForAutoDelegtion(excludeFromAutoDelegation), epoch.Seq+1)

	for p, state := range e.partyDelegationState {
		if _, ok := e.autoDelegationMode[p]; !ok {
			if balance, err := e.stakingAccounts.GetAvailableBalance(p); err == nil {
				if state.totalDelegated.ToDecimal().Div(balance.ToDecimal()).GreaterThanOrEqual(minRatioForAutoDelegation) {
					e.autoDelegationMode[p] = struct{}{}
				}
			}
		}
	}
	return stateForRewards
}

// sendDelegatedBalanceEvent emits an event with the delegation balance for the given epoch.
func (e *Engine) sendDelegatedBalanceEvent(ctx context.Context, party, nodeID string, seq uint64, amt *num.Uint) {
	if amt == nil {
		e.broker.Send(events.NewDelegationBalance(ctx, party, nodeID, num.UintZero(), num.NewUint(seq).String()))
	} else {
		e.broker.Send(events.NewDelegationBalance(ctx, party, nodeID, amt.Clone(), num.NewUint(seq).String()))
	}
}

// decrease the delegation balance fire an event and cleanup if requested.
func (e *Engine) decreaseBalanceAndFireEvent(ctx context.Context, party, nodeID string, amt *num.Uint, epoch uint64, delegationState map[string]*partyDelegation, cleanup, fireEvent bool) {
	if _, ok := delegationState[party]; !ok {
		return
	}
	partyState := delegationState[party]
	if partyState.totalDelegated.GT(amt) {
		partyState.totalDelegated.Sub(partyState.totalDelegated, amt)
	} else {
		partyState.totalDelegated = num.UintZero()
	}

	if nodeAmt, ok := partyState.nodeToAmount[nodeID]; ok {
		if nodeAmt.GT(amt) {
			partyState.nodeToAmount[nodeID].Sub(nodeAmt, amt)
		} else {
			partyState.nodeToAmount[nodeID] = num.UintZero()
		}
		if fireEvent {
			e.sendDelegatedBalanceEvent(ctx, party, nodeID, epoch, partyState.nodeToAmount[nodeID])
		}
		if cleanup && partyState.nodeToAmount[nodeID].IsZero() {
			delete(partyState.nodeToAmount, nodeID)
		}
	}

	if cleanup && partyState.totalDelegated.IsZero() {
		delete(delegationState, party)
	}
}

// sort node IDs for deterministic processing.
func (e *Engine) sortNodes(nodes map[string]*num.Uint) []string {
	nodeIDs := make([]string, 0, len(nodes))
	for nodeID := range nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}

	// sort the parties for deterministic handling
	sort.Strings(nodeIDs)
	return nodeIDs
}

func (e *Engine) sortParties(delegation map[string]*partyDelegation) []string {
	parties := make([]string, 0, len(delegation))
	for party := range delegation {
		parties = append(parties, party)
	}

	// sort the parties for deterministic handling
	sort.Strings(parties)
	return parties
}

func (e *Engine) stakeInRangeFunc(from, to time.Time) func(string) (*num.Uint, error) {
	return func(party string) (*num.Uint, error) {
		return e.stakingAccounts.GetAvailableBalanceInRange(party, from, to)
	}
}

// take a copy of the next epoch delegation ignoring delegations that have been zero for the currend and next epoch.
func (e *Engine) prepareNextEpochDelegationState() map[string]*partyDelegation {
	nextEpoch := make(map[string]*partyDelegation, len(e.nextPartyDelegationState))
	for party, partyDS := range e.nextPartyDelegationState {
		nextEpoch[party] = &partyDelegation{
			totalDelegated: partyDS.totalDelegated.Clone(),
			nodeToAmount:   make(map[string]*num.Uint, len(partyDS.nodeToAmount)),
		}
		for n, amt := range partyDS.nodeToAmount {
			if amt.IsZero() {
				// check the balance in the previous epoch - if it was there and was non zero keep it, otherwise it means it hasn't changed so we can drop
				if pds, ok := e.partyDelegationState[party]; ok {
					if prevAmt, ok := pds.nodeToAmount[n]; ok && !prevAmt.IsZero() {
						nextEpoch[party].nodeToAmount[n] = amt.Clone()
					}
				}
			} else {
				nextEpoch[party].nodeToAmount[n] = amt.Clone()
			}
		}
	}
	return nextEpoch
}

// eligiblePartiesForAutoDelegtion calculates how much is available for auto delegation in parties that have qualifies for auto delegation
// and have not done any manual actions during the past epoch and have any active delegations and have available balance.
func (e *Engine) eligiblePartiesForAutoDelegtion(exclude map[string]struct{}) map[string]*num.Uint {
	partyToAvailableBalance := map[string]*num.Uint{}
	for party := range e.autoDelegationMode {
		// if the party has no delegation we can't auto delegate
		if _, ok := e.partyDelegationState[party]; !ok {
			continue
		}

		if _, ok := exclude[party]; ok {
			continue
		}

		// check if they have balance
		balance, err := e.stakingAccounts.GetAvailableBalance(party)
		if err != nil {
			continue
		}

		// check how much they already have delegated off the staking account balance
		delegated := e.partyDelegationState[party].totalDelegated
		if delegated.GTE(balance) {
			continue
		}

		// calculate the available balance
		available := num.UintZero().Sub(balance, delegated)
		if !available.IsZero() {
			partyToAvailableBalance[party] = available
		}
	}
	return partyToAvailableBalance
}

// processAutoDelegation takes a slice of parties which are known to be eligible for auto delegation and attempts to distribute their available
// undelegated stake proportionally across the nodes to which it already delegated to.
// It respects the max delegation per validator, and if the node does not accept any more stake it will not try to delegate it to other nodes.
func (e *Engine) processAutoDelegation(ctx context.Context, partyToAvailableBalance map[string]*num.Uint, seq uint64) {
	parties := make([]string, 0, len(partyToAvailableBalance))
	for p := range partyToAvailableBalance {
		parties = append(parties, p)
	}
	sort.Strings(parties)

	for _, p := range parties {
		totalDelegation := e.partyDelegationState[p].totalDelegated.ToDecimal()
		balanceDec := partyToAvailableBalance[p].ToDecimal()
		nodes := e.sortNodes(e.partyDelegationState[p].nodeToAmount)

		for _, n := range nodes {
			nodeBalance := e.partyDelegationState[p].nodeToAmount[n]
			ratio := nodeBalance.ToDecimal().Div(totalDelegation)
			delegationToNodeN, _ := num.UintFromDecimal(ratio.Mul(balanceDec))

			if !delegationToNodeN.IsZero() {
				e.partyDelegationState[p].totalDelegated.AddSum(delegationToNodeN)
				e.partyDelegationState[p].nodeToAmount[n].AddSum(delegationToNodeN)
				e.sendDelegatedBalanceEvent(ctx, p, n, seq, e.partyDelegationState[p].nodeToAmount[n])
				e.nextPartyDelegationState[p].totalDelegated.AddSum(delegationToNodeN)
				e.nextPartyDelegationState[p].nodeToAmount[n].AddSum(delegationToNodeN)
			}
		}
	}
}

// GetValidatorData returns the current state of the delegation per node.
func (e *Engine) GetValidatorData() []*types.ValidatorData {
	return e.getValidatorData()
}

// returns the current state of the delegation per node.
func (e *Engine) getValidatorData() []*types.ValidatorData {
	validatorNodes := e.topology.AllNodeIDs()
	validatorData := make(map[string]*types.ValidatorData, len(validatorNodes))

	for _, vn := range validatorNodes {
		validatorData[vn] = &types.ValidatorData{
			NodeID:            vn,
			PubKey:            e.topology.Get(vn).VegaPubKey,
			Delegators:        map[string]*num.Uint{},
			SelfStake:         num.UintZero(),
			StakeByDelegators: num.UintZero(),
			TmPubKey:          e.topology.Get(vn).TmPubKey,
		}
	}

	for party, partyDS := range e.partyDelegationState {
		for node, amt := range partyDS.nodeToAmount {
			vn := validatorData[node]
			if party == vn.PubKey {
				vn.SelfStake = amt.Clone()
			} else {
				vn.Delegators[party] = amt.Clone()
				vn.StakeByDelegators.AddSum(amt)
			}
		}
	}

	validators := make([]*types.ValidatorData, 0, len(validatorNodes))
	sort.Strings(validatorNodes)
	for _, v := range validatorNodes {
		validators = append(validators, validatorData[v])
	}

	return validators
}
