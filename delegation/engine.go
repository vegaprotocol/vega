package delegation

import (
	"context"
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var minVal, _ = num.DecimalFromString("5.0")

var (
	// ErrPartyHasNoStakingAccount is returned when the staking account for the party cannot be found
	ErrPartyHasNoStakingAccount = errors.New("cannot find staking account for the party")
	// ErrInvalidNodeID is returned when the node id passed for delegation/undelegation is not a validator node identifier
	ErrInvalidNodeID = errors.New("invalid node ID")
	// ErrInsufficientBalanceForDelegation is returned when the balance in the staking account is insufficient to cover all committed and pending delegations
	ErrInsufficientBalanceForDelegation = errors.New("insufficient balance for delegation")
	// ErrIncorrectTokenAmountForUndelegation is returned when the amount to undelegation doesn't match the delegation balance (pending + committed) for the party and validator
	ErrIncorrectTokenAmountForUndelegation = errors.New("incorrect token amount for undelegation")
	// ErrAmountLTMinAmountForDelegation is returned when the amount to delegate to a node is lower than the minimum allowed amount from network params
	ErrAmountLTMinAmountForDelegation = errors.New("delegation amount is lower than the minimum amount for delegation for a validator")
)

// ValidatorTopology represents the topology of validators and can check if a given node is a validator
type ValidatorTopology interface {
	IsValidatorNode(nodeID string) bool
	AllPubKeys() []string
}

// Broker send events
// we no longer need to generate this mock here, we can use the broker/mocks package instead
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// StakingAccounts provides access to the staking balance of a given party now and within a duration of an epoch
type StakingAccounts interface {
	GetAvailableBalance(party string) (*num.Uint, error)
	GetAvailableBalanceInRange(party string, from, to time.Time) (*num.Uint, error)
}

type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

// validator delegation state - updated at the end of each epoch
type validatorDelegation struct {
	nodeID         string               // node id
	partyToAmount  map[string]*num.Uint // party -> delegated amount
	totalDelegated *num.Uint            // the total amount delegates by parties
}

// party delegation state - how much is delegated by the party to each validator and in total
type partyDelegation struct {
	party          string               // party ID
	nodeToAmount   map[string]*num.Uint // nodeID -> delegated amount
	totalDelegated *num.Uint            // total amount delegated by party
}

// party delegation state
type pendingPartyDelegation struct {
	party                  string
	nodeToDelegateAmount   map[string]*num.Uint
	nodeToUndelegateAmount map[string]*num.Uint
	totalDelegation        *num.Uint
	totalUndelegation      *num.Uint
}

// Engine is handling the delegations balances from parties to validators
// The delegation engine is designed in the following way with the following assumptions:
// 1. during epoch it is called with delegation requests that are added to a pending data structure and only applied at the end of the epoch
// 2. At the end of the epoch - this is not necessarily at the end of the epoch but rather when told to process later than the end of epoch- the engine is called and does the following:
// 2.1 updates the delegated balances to match the epoch's staking account balance for each party such that if a party withdrew from their
//     staking account during the epoch it will not count for them for rewarding
// 2.2 capture the state after 2.1 to be returned to the rewarding engine
// 2.3 process all pending delegations
type Engine struct {
	log                  *logging.Logger
	config               Config
	broker               Broker
	topology             ValidatorTopology                             // an interface to the topoology to interact with validator nodes if needed
	stakingAccounts      StakingAccounts                               // an interface to the staking account for getting party balances
	nodeDelegationState  map[string]*validatorDelegation               // validator to active delegations
	partyDelegationState map[string]*partyDelegation                   // party to active delegations
	pendingState         map[uint64]map[string]*pendingPartyDelegation // epoch seq -> pending delegations/undelegations by party
	minDelegationAmount  *num.Uint                                     // min delegation amount per delegation request
	currentEpoch         types.Epoch                                   // the current epoch for pending delegations
	compLevel            num.Decimal
}

//New instantiate a new delegation engine
func New(log *logging.Logger, config Config, broker Broker, topology ValidatorTopology, stakingAccounts StakingAccounts, epochEngine EpochEngine) *Engine {

	e := &Engine{
		config:               config,
		log:                  log.Named(namedLogger),
		broker:               broker,
		topology:             topology,
		stakingAccounts:      stakingAccounts,
		nodeDelegationState:  map[string]*validatorDelegation{},
		partyDelegationState: map[string]*partyDelegation{},
		pendingState:         map[uint64]map[string]*pendingPartyDelegation{},
	}

	// register for epoch notifications
	epochEngine.NotifyOnEpoch(e.onEpochEvent)

	return e
}

//OnCompLevelChanged updates the network parameter for competitionLevel
func (e *Engine) OnCompLevelChanged(ctx context.Context, compLevel float64) error {
	e.compLevel = num.DecimalFromFloat(compLevel)
	return nil
}

//OnMinAmountChanged updates the network parameter for minDelegationAmount
func (e *Engine) OnMinAmountChanged(ctx context.Context, minAmount num.Decimal) error {
	e.minDelegationAmount, _ = num.UintFromDecimal(minAmount)
	return nil
}

//update the current epoch at which current pending delegations are recorded
//regardless if the event is start or stop of the epoch. the sequence is what identifies the epoch
func (e *Engine) onEpochEvent(ctx context.Context, epoch types.Epoch) {
	e.currentEpoch = epoch
}

//ProcessEpochDelegations updates the delegation engine state at the end of a given epoch and returns the validation-delegation data for rewarding for that epoch
// step 1: process delegation data for the epoch - undelegate if the balance of the staking account doesn't cover all delegations
// step 2: capture validator delegation data to be returned
// step 3: apply pending undelegations
// step 4: apply pending delegations
// epoch here is the epoch that ended
func (e *Engine) ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData {
	if e.log.IsDebug() {
		e.log.Debug("on epoch end:", logging.Time("start", epoch.StartTime), logging.Time("end", epoch.EndTime))
	}

	partiesForEvents := map[string]map[string]string{}

	if pendingForEpoch, ok := e.pendingState[epoch.Seq]; ok {
		for party, nodes := range pendingForEpoch {
			partiesForEvents[party] = map[string]string{}
			for node := range nodes.nodeToDelegateAmount {
				partiesForEvents[party][node] = node
			}
			for node := range nodes.nodeToUndelegateAmount {
				partiesForEvents[party][node] = node
			}
		}
	}
	for party, nodes := range e.partyDelegationState {
		if _, ok := partiesForEvents[party]; !ok {
			partiesForEvents[party] = map[string]string{}
		}
		for node := range nodes.nodeToAmount {
			partiesForEvents[party][node] = node
		}
	}

	partiesForEventsSlice := make([]string, 0, len(partiesForEvents))
	for party := range partiesForEvents {
		partiesForEventsSlice = append(partiesForEventsSlice, party)
	}
	sort.Strings(partiesForEventsSlice)

	e.preprocessEpochForRewarding(ctx, epoch)
	stateForRewards := e.getValidatorData()
	e.processPending(ctx, epoch)

	// we need to send an event for the following epoch
	for _, party := range partiesForEventsSlice {
		nodesForParty := partiesForEvents[party]
		nodesSlice := make([]string, 0, len(nodesForParty))
		for node := range nodesForParty {
			nodesSlice = append(nodesSlice, node)
		}
		sort.Strings(nodesSlice)
		for _, node := range nodesSlice {
			e.sendNextEpochBalanceEvent(ctx, party, node, epoch.Seq)
		}

	}

	return stateForRewards
}

//Delegate increases the pending delegation balance and potentially decreases the pending undelegation balance for a given validator node
func (e *Engine) Delegate(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	amt := amount.Clone()

	// check if the node is a validator node
	if !e.topology.IsValidatorNode(nodeID) {
		return ErrInvalidNodeID
	}

	// check if the delegator has a staking account
	partyBalance, err := e.stakingAccounts.GetAvailableBalance(party)
	if err != nil {
		return ErrPartyHasNoStakingAccount
	}

	if amt.LT(e.minDelegationAmount) {
		return ErrAmountLTMinAmountForDelegation
	}

	// get the pending state for the current epoch - there may be more than one unprocessed pending epochs depending
	pendingForEpoch, ok := e.pendingState[e.currentEpoch.Seq]
	if !ok {
		pendingForEpoch = map[string]*pendingPartyDelegation{}
		e.pendingState[e.currentEpoch.Seq] = pendingForEpoch
	}

	// check if the delegator has sufficient balance in their staking account including all pending and committed delegations and undelegations
	// this is basically just fail fast - the delegation may still fail
	currentPendingPartyDelegation, ok := pendingForEpoch[party]
	if !ok {
		pendingForEpoch[party] = &pendingPartyDelegation{
			party:                  party,
			totalDelegation:        num.Zero(),
			totalUndelegation:      num.Zero(),
			nodeToUndelegateAmount: map[string]*num.Uint{},
			nodeToDelegateAmount:   map[string]*num.Uint{},
		}
		currentPendingPartyDelegation = pendingForEpoch[party]
	}
	partyDelegation, ok := e.partyDelegationState[party]
	partyDelegationBalance := num.Zero()
	if ok {
		partyDelegationBalance = partyDelegation.totalDelegated
	}

	// if the party withdrew from their account and now don't have sufficient cover for their current delegation, prevent them from further delgations
	// no need to immediately undelegate because this will be handled at epoch end
	if partyBalance.LTE(partyDelegationBalance) {
		return ErrInsufficientBalanceForDelegation
	}

	// subrtact the committed delegation balance and apply pending if any

	balanceAvailableForDelegation := num.Zero().Sub(partyBalance, partyDelegationBalance)
	partyPendingDelegation := currentPendingPartyDelegation.totalDelegation
	partyPendingUndelegation := currentPendingPartyDelegation.totalUndelegation

	// add pending undelegations to available balance
	if !partyPendingUndelegation.IsZero() {
		balanceAvailableForDelegation.AddSum(partyPendingUndelegation)
	}
	// subtract pending delegations from available balance
	if !partyPendingDelegation.IsZero() {
		// if there's somehow more pending than available for delegation due to withdrawls return error
		if partyPendingDelegation.GT(balanceAvailableForDelegation) {
			return ErrInsufficientBalanceForDelegation
		}
		balanceAvailableForDelegation = num.Zero().Sub(balanceAvailableForDelegation, partyPendingDelegation)
	}

	// if the balance with committed and pending delegations/undelegations is insufficient to satisfy the delegation return error
	if balanceAvailableForDelegation.LT(amt) {
		return ErrInsufficientBalanceForDelegation
	}

	// all good add to pending delegation
	remainingBalanceForDelegate := amt

	partyPendingUndelegationForNode, udok := currentPendingPartyDelegation.nodeToUndelegateAmount[nodeID]
	partyPendingDelegationForNode, dok := currentPendingPartyDelegation.nodeToDelegateAmount[nodeID]

	if udok { // we have undelegates which we can counter
		if remainingBalanceForDelegate.GTE(partyPendingUndelegationForNode) {
			// the delegation amount is greater than or equal to the undelegated amount, we can clear the whole undelegation and leave the remaining delegation
			remainingBalanceForDelegate = num.Zero().Sub(remainingBalanceForDelegate, partyPendingUndelegationForNode)
			delete(currentPendingPartyDelegation.nodeToUndelegateAmount, nodeID)
			currentPendingPartyDelegation.totalUndelegation = num.Zero().Sub(currentPendingPartyDelegation.totalUndelegation, partyPendingUndelegationForNode)
			currentPendingPartyDelegation.totalDelegation = num.Sum(currentPendingPartyDelegation.totalDelegation, remainingBalanceForDelegate)

			if !remainingBalanceForDelegate.IsZero() {
				currentPendingPartyDelegation.nodeToDelegateAmount[nodeID] = remainingBalanceForDelegate
			} else {
				delete(currentPendingPartyDelegation.nodeToDelegateAmount, nodeID)
				if currentPendingPartyDelegation.totalUndelegation.IsZero() && currentPendingPartyDelegation.totalDelegation.IsZero() {
					delete(pendingForEpoch, party)
				}
			}
		} else {
			// the delegation amount is lower than the pending undelegate amount - we can just adjust the undelegate amount
			updatedUndelegateAmout := num.Zero().Sub(partyPendingUndelegationForNode, remainingBalanceForDelegate)
			currentPendingPartyDelegation.nodeToUndelegateAmount[nodeID] = updatedUndelegateAmout
			currentPendingPartyDelegation.totalUndelegation = num.Zero().Sub(currentPendingPartyDelegation.totalUndelegation, remainingBalanceForDelegate)

		}
	} else {
		// there are no pending undelegations we can just update the pending delegation
		if !dok {
			partyPendingDelegationForNode = num.Zero()
		}
		currentPendingPartyDelegation.nodeToDelegateAmount[nodeID] = num.Sum(partyPendingDelegationForNode, remainingBalanceForDelegate)
		currentPendingPartyDelegation.totalDelegation = num.Sum(currentPendingPartyDelegation.totalDelegation, remainingBalanceForDelegate)
	}

	e.sendNextEpochBalanceEvent(ctx, party, nodeID, e.currentEpoch.Seq)

	return nil
}

//UndelegateAtEndOfEpoch increases the pending undelegation balance and potentially decreases the pending delegation balance for a given validator node and party
func (e *Engine) UndelegateAtEndOfEpoch(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	amt := amount.Clone()

	pendingForEpoch, ok := e.pendingState[e.currentEpoch.Seq]
	if !ok {
		pendingForEpoch = map[string]*pendingPartyDelegation{}
		e.pendingState[e.currentEpoch.Seq] = pendingForEpoch
	}

	if amt.IsZero() {
		// calculate how much we have available for undelegation including pending and committed
		availableForUndelegationInPending := num.Zero()
		if pendingState, ok := pendingForEpoch[party]; ok {
			if nodeDelegation, ok := pendingState.nodeToDelegateAmount[nodeID]; ok {
				availableForUndelegationInPending = num.Sum(availableForUndelegationInPending, nodeDelegation)
			}
		}
		availableForUndelegationInActive := num.Zero()
		if partyDelegation, ok := e.partyDelegationState[party]; ok {
			if nodeDelegation, ok := partyDelegation.nodeToAmount[nodeID]; ok {
				availableForUndelegationInActive = num.Sum(availableForUndelegationInActive, nodeDelegation)
			}
		}
		amt = amt.AddSum(availableForUndelegationInPending, availableForUndelegationInActive)
	}

	// check if the node is a validator node
	if e.topology == nil || !e.topology.IsValidatorNode(nodeID) {
		return ErrInvalidNodeID
	}

	// get the delegated balance for the given node
	validatorState, ok := e.nodeDelegationState[nodeID]
	partyDelegatedToNodeAmount := num.Zero()
	if ok {
		partyDelegatedToNodeAmount, ok = validatorState.partyToAmount[party]
		if !ok {
			partyDelegatedToNodeAmount = num.Zero()
		}
	}

	pendingDelegateToNodeAmount := num.Zero()
	pendingUndelegateToNodeAmount := num.Zero()

	// check if there is anything pending
	currentPendingPartyDelegation, ok := pendingForEpoch[party]
	if ok {
		pendingDelegateToNodeAmount, ok = currentPendingPartyDelegation.nodeToDelegateAmount[nodeID]
		if !ok {
			pendingDelegateToNodeAmount = num.Zero()
		}
		pendingUndelegateToNodeAmount, ok = currentPendingPartyDelegation.nodeToUndelegateAmount[nodeID]
		if !ok {
			pendingUndelegateToNodeAmount = num.Zero()
		}
	} else {
		// if there isn't yet a pending state, construct it here
		currentPendingPartyDelegation = &pendingPartyDelegation{
			party:                  party,
			totalDelegation:        num.Zero(),
			totalUndelegation:      num.Zero(),
			nodeToUndelegateAmount: map[string]*num.Uint{},
			nodeToDelegateAmount:   map[string]*num.Uint{},
		}
	}

	totalDelegationBalance := num.Sum(partyDelegatedToNodeAmount, pendingDelegateToNodeAmount)
	totalDelegationBalance = num.Zero().Sub(totalDelegationBalance, pendingUndelegateToNodeAmount)

	// if the amount is greater than the available balance to undelegate return error
	if amt.GT(totalDelegationBalance) {
		return ErrIncorrectTokenAmountForUndelegation
	}

	remainingBalanceForUndelegate := amt

	if !pendingDelegateToNodeAmount.IsZero() { // we have delegates which we can counter
		if remainingBalanceForUndelegate.GTE(pendingDelegateToNodeAmount) {
			// the undelegation amount is greater than or equal to the delegated amount, we can clear the whole delegation and leave the remaining undelegation
			remainingBalanceForUndelegate = num.Zero().Sub(remainingBalanceForUndelegate, pendingDelegateToNodeAmount)
			currentPendingPartyDelegation.totalDelegation = num.Zero().Sub(currentPendingPartyDelegation.totalDelegation, pendingDelegateToNodeAmount)
			currentPendingPartyDelegation.totalUndelegation = num.Sum(currentPendingPartyDelegation.totalUndelegation, remainingBalanceForUndelegate)

			delete(currentPendingPartyDelegation.nodeToDelegateAmount, nodeID)
			if !remainingBalanceForUndelegate.IsZero() {
				currentPendingPartyDelegation.nodeToUndelegateAmount[nodeID] = remainingBalanceForUndelegate
			} else {
				delete(currentPendingPartyDelegation.nodeToUndelegateAmount, nodeID)
				if currentPendingPartyDelegation.totalUndelegation.IsZero() && currentPendingPartyDelegation.totalDelegation.IsZero() {
					delete(pendingForEpoch, party)
				}
			}
		} else {
			// the undelegation amount is lower than the pending delegate amount - we can just adjust the delegate amount
			updatedDelegateAmount := num.Zero().Sub(pendingDelegateToNodeAmount, remainingBalanceForUndelegate)
			currentPendingPartyDelegation.totalDelegation = num.Zero().Sub(currentPendingPartyDelegation.totalDelegation, remainingBalanceForUndelegate)
			currentPendingPartyDelegation.nodeToDelegateAmount[nodeID] = updatedDelegateAmount
		}
	} else {
		// there are no pending delegations we can just update the pending undelegation
		currentPendingPartyDelegation.nodeToUndelegateAmount[nodeID] = num.Sum(pendingUndelegateToNodeAmount, remainingBalanceForUndelegate)
		currentPendingPartyDelegation.totalUndelegation = num.Sum(currentPendingPartyDelegation.totalUndelegation, remainingBalanceForUndelegate)
	}

	_, ok = pendingForEpoch[party]
	// if there was no previous undelegation and we ended up undelegating, add to state
	if !ok && !currentPendingPartyDelegation.totalUndelegation.IsZero() {
		pendingForEpoch[party] = currentPendingPartyDelegation
	}

	e.sendNextEpochBalanceEvent(ctx, party, nodeID, e.currentEpoch.Seq)
	return nil
}

//UndelegateNow changes the balance of delegation immediately without waiting for the end of the epoch
// if possible it removed balance from pending delegated, if not enough it removes balance from the current epoch delegated amount
func (e *Engine) UndelegateNow(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	// first check available balance for undelegation and error if the requested amount is greater than
	availableForUndelegationInPending := num.Zero()

	// check if we have any pending in any unprocessed epoch
	pendingEpochs := []uint64{}
	for epoch, pendingForEpoch := range e.pendingState {
		if pendingState, ok := pendingForEpoch[party]; ok {
			pendingEpochs = append(pendingEpochs, epoch)
			if nodeDelegation, ok := pendingState.nodeToDelegateAmount[nodeID]; ok {
				availableForUndelegationInPending = num.Sum(availableForUndelegationInPending, nodeDelegation)
			}
		}
	}
	// sort pendingEpochs descending so we can start from the last pending epoch
	sort.Slice(pendingEpochs, func(i, j int) bool { return pendingEpochs[i] > pendingEpochs[j] })

	availableForUndelegationInActive := num.Zero()
	if partyDelegation, ok := e.partyDelegationState[party]; ok {
		if nodeDelegation, ok := partyDelegation.nodeToAmount[nodeID]; ok {
			availableForUndelegationInActive = num.Sum(availableForUndelegationInActive, nodeDelegation)
		}
	}

	totalAvailableForUndelegation := num.Sum(availableForUndelegationInPending, availableForUndelegationInActive)

	// if the party passes 0 they want to undelegate all
	var amt = amount.Clone()
	if amt.IsZero() {
		amt = totalAvailableForUndelegation.Clone()
	}

	if amt.GT(totalAvailableForUndelegation) {
		return ErrIncorrectTokenAmountForUndelegation
	}

	// strart with undelegating from pending, if not enough go to active
	if !availableForUndelegationInPending.IsZero() {
		for _, epoch := range pendingEpochs {
			pendingForEpoch := e.pendingState[epoch]
			pendingState := pendingForEpoch[party]
			if amt.LTE(availableForUndelegationInPending) {
				pendingState.nodeToDelegateAmount[nodeID] = num.Zero().Sub(availableForUndelegationInPending, amt)
				pendingState.totalDelegation = num.Zero().Sub(pendingState.totalDelegation, amt)
				if pendingState.nodeToDelegateAmount[nodeID].IsZero() {
					delete(pendingState.nodeToDelegateAmount, nodeID)
				}
				amt = num.Zero()
			} else {
				// we don't have enough delegation to cover for the undelegate request
				pendingState.totalDelegation = num.Zero().Sub(pendingState.totalDelegation, availableForUndelegationInPending)
				delete(pendingState.nodeToDelegateAmount, nodeID)
				amt = amt.Sub(amt, availableForUndelegationInPending)
			}

			if pendingState.totalDelegation.IsZero() && pendingState.totalUndelegation.IsZero() {
				delete(pendingForEpoch, party)
			}
		}
	}
	// if there's still some balance to undelegate we go to the delegated state
	if !amt.IsZero() {
		partyDelegation := e.partyDelegationState[party]
		partyDelegation.totalDelegated = num.Zero().Sub(partyDelegation.totalDelegated, amt)
		partyDelegation.nodeToAmount[nodeID] = num.Zero().Sub(partyDelegation.nodeToAmount[nodeID], amt)
		if partyDelegation.nodeToAmount[nodeID].IsZero() {
			delete(partyDelegation.nodeToAmount, nodeID)
		}
		if partyDelegation.totalDelegated.IsZero() {
			delete(e.partyDelegationState, party)
		}
		nodeDelegation, ok := e.nodeDelegationState[nodeID]
		if !ok {
			e.log.Panic("party and node delegation state disagree")
		}
		nodeDelegation.totalDelegated = num.Zero().Sub(nodeDelegation.totalDelegated, amt)
		nodeDelegation.partyToAmount[party] = num.Zero().Sub(nodeDelegation.partyToAmount[party], amt)
		if nodeDelegation.partyToAmount[party].IsZero() {
			delete(nodeDelegation.partyToAmount, party)
		}
		if nodeDelegation.totalDelegated.IsZero() {
			delete(e.nodeDelegationState, nodeID)
		}

	}
	e.sendDelegatedBalanceEvent(ctx, party, nodeID, e.currentEpoch.Seq)
	e.sendNextEpochBalanceEvent(ctx, party, nodeID, e.currentEpoch.Seq)
	return nil
}

// sends the expected balance for the next epoch
func (e *Engine) sendNextEpochBalanceEvent(ctx context.Context, party, nodeID string, seq uint64) {
	pendingState, ok := e.pendingState[seq][party]

	pendingDelegated := num.Zero()
	pendingUndelegated := num.Zero()
	var dok, udok bool

	if ok {
		pendingDelegated, dok = pendingState.nodeToDelegateAmount[nodeID]
		if !dok {
			pendingDelegated = num.Zero()
		}
		pendingUndelegated, udok = pendingState.nodeToUndelegateAmount[nodeID]
		if !udok {
			pendingUndelegated = num.Zero()
		}
	}
	delegatedToNode := num.Zero()
	if currentlyInPlay, ok := e.partyDelegationState[party]; ok {
		if nodeDelegation, ok := currentlyInPlay.nodeToAmount[nodeID]; ok {
			delegatedToNode = nodeDelegation
		}
	}

	amt := num.Zero().Sub(num.Sum(delegatedToNode, pendingDelegated), pendingUndelegated)
	effEpoch := seq + 1
	e.broker.Send(events.NewDelegationBalance(ctx, party, nodeID, amt, num.NewUint(effEpoch).String()))
}

func (e *Engine) sendDelegatedBalanceEvent(ctx context.Context, party, nodeID string, seq uint64) {
	delegated, ok := e.partyDelegationState[party]

	if ok {
		amt, ok := delegated.nodeToAmount[nodeID]
		if !ok {
			amt = num.Zero()
		}
		e.broker.Send(events.NewDelegationBalance(ctx, party, nodeID, amt, num.NewUint(seq).String()))
		return
	}

	e.broker.Send(events.NewDelegationBalance(ctx, party, nodeID, num.Zero(), num.NewUint(seq).String()))
}

func (e *Engine) decreaseDelegationAmountBy(party, nodeID string, amt *num.Uint) {
	partyDelegation := e.partyDelegationState[party]
	nodeDelegation := e.nodeDelegationState[nodeID]

	// update the balance for the validator for the party
	partyDelegation.nodeToAmount[nodeID] = num.Zero().Sub(partyDelegation.nodeToAmount[nodeID], amt)
	partyDelegation.totalDelegated = num.Zero().Sub(partyDelegation.totalDelegated, amt)

	// if there's no more delegations, remove the entry for the nodeID
	if partyDelegation.nodeToAmount[nodeID].IsZero() {
		delete(partyDelegation.nodeToAmount, nodeID)
	}
	if partyDelegation.totalDelegated.IsZero() {
		delete(e.partyDelegationState, party)
	}

	// update the balance for the party for the validator
	nodeDelegation.partyToAmount[party] = num.Zero().Sub(nodeDelegation.partyToAmount[party], amt)
	nodeDelegation.totalDelegated = num.Zero().Sub(nodeDelegation.totalDelegated, amt)

	// if there's no more delegations, remove the entry for the nodeID
	if nodeDelegation.partyToAmount[party].IsZero() {
		delete(nodeDelegation.partyToAmount, party)
	}
	if nodeDelegation.totalDelegated.IsZero() {
		delete(e.nodeDelegationState, nodeID)
	}

}

// sort node IDs for deterministic processing
func (e *Engine) sortNodes(nodes map[string]*num.Uint) []string {
	nodeIDs := make([]string, 0, len(nodes))
	for nodeID := range nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}

	// sort the parties for deterministic handling
	sort.Strings(nodeIDs)
	return nodeIDs
}

// preprocessEpoch is called at the end of an epoch and updates the state to be returned for rewarding calculation
// check balance for the epoch duration and undelegate if delegations don't have sufficient cover
// the state of the engine by the end of this method reflects the state to be used for reward engine
func (e *Engine) preprocessEpochForRewarding(ctx context.Context, epoch types.Epoch) {
	parties := make([]string, 0, len(e.partyDelegationState))
	for party := range e.partyDelegationState {
		parties = append(parties, party)
	}

	// sort the parties for deterministic handling
	sort.Strings(parties)

	// for all parties with delegations in the ended epoch
	for _, party := range parties {
		partyDelegation := e.partyDelegationState[party]

		// get the party stake balance for the epoch
		stakeBalance, err := e.stakingAccounts.GetAvailableBalanceInRange(party, epoch.StartTime, epoch.EndTime)
		if err != nil {
			e.log.Error("Failed to get available balance in range", logging.Error(err))
			continue
		}

		// if the stake covers the total delegated balance nothing to do further for the party
		if stakeBalance.GTE(partyDelegation.totalDelegated) {
			continue
		}

		// if the stake account balance for the epoch is less than the delegated balance - we need to undelegate the difference
		// this will be done evenly as much as possible between all validators with delegation from the party
		remainingBalanceToUndelegate := num.Zero().Sub(partyDelegation.totalDelegated, stakeBalance)

		totalTaken := num.Zero()

		nodeIDs := e.sortNodes(partyDelegation.nodeToAmount)

		// undelegate proportionally across delegated validator nodes
		totalDeletation := partyDelegation.totalDelegated.Clone()
		for _, nodeID := range nodeIDs {
			balance := partyDelegation.nodeToAmount[nodeID]
			balanceToTake := num.Zero().Mul(balance, remainingBalanceToUndelegate)
			balanceToTake = num.Zero().Div(balanceToTake, totalDeletation)

			if balanceToTake.IsZero() {
				continue
			}

			e.decreaseDelegationAmountBy(party, nodeID, balanceToTake)
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
					e.decreaseDelegationAmountBy(party, nodeID, num.NewUint(1))
					totalTaken = num.Sum(totalTaken, num.NewUint(1))
				}
			}
		}

		if len(partyDelegation.nodeToAmount) == 0 {
			delete(e.partyDelegationState, party)
		}

		for _, nodeID := range nodeIDs {
			e.sendDelegatedBalanceEvent(ctx, party, nodeID, epoch.Seq)
		}
	}
}

// calculate the total number of tokens (a rough estimate) and the number of nodes
func (e *Engine) calcTotalDelegatedTokens(epochSeq uint64) *num.Uint {
	totalDelegatedTokens := num.Zero()
	for _, nodeDel := range e.nodeDelegationState {
		totalDelegatedTokens.AddSum(nodeDel.totalDelegated)
	}
	if pendingForEpoch, ok := e.pendingState[epochSeq]; ok {
		for _, pendingDel := range pendingForEpoch {
			totalDelegatedTokens = totalDelegatedTokens.Sub(totalDelegatedTokens.AddSum(pendingDel.totalDelegation), pendingDel.totalUndelegation)
		}
	}
	return totalDelegatedTokens
}

func (e *Engine) calcMaxDelegatableTokens(totalTokens *num.Uint, numVal num.Decimal) *num.Uint {
	a := num.MaxD(minVal, numVal.Div(e.compLevel))

	res, _ := num.UintFromDecimal(totalTokens.ToDecimal().Div(a))
	return res
}

// process pending delegations and undelegations at the end of the epoch and clear the delegation/undelegation maps at the end
func (e *Engine) processPending(ctx context.Context, epoch types.Epoch) {
	pendingForEpoch, ok := e.pendingState[epoch.Seq]
	if !ok {
		// no pending for epoch
		return
	}

	parties := make([]string, 0, len(pendingForEpoch))
	partyNodes := map[string][]string{}
	for party, state := range pendingForEpoch {
		parties = append(parties, party)
		nodes := map[string]bool{}
		for node := range state.nodeToDelegateAmount {
			nodes[node] = true
		}
		for node := range state.nodeToUndelegateAmount {
			nodes[node] = true
		}
		var nodesSlice []string
		for node := range nodes {
			nodesSlice = append(nodesSlice, node)
		}
		sort.Strings(nodesSlice)
		partyNodes[party] = nodesSlice
	}

	// sort the parties for deterministic handling
	sort.Strings(parties)
	// calculate the total number of tokens (a rough estimate)
	totalTokens := e.calcTotalDelegatedTokens(epoch.Seq)
	// calculate the max for the next epoch
	numVal := len(e.topology.AllPubKeys())
	maxStakePerValidator := e.calcMaxDelegatableTokens(totalTokens, num.DecimalFromInt64(int64(numVal)))

	// read the delegation min amount network param
	e.processPendingUndelegations(parties, epoch)
	e.processPendingDelegations(parties, maxStakePerValidator, epoch)

	delete(e.pendingState, epoch.Seq)
}

// process pending undelegations for all parties
func (e *Engine) processPendingUndelegations(parties []string, epoch types.Epoch) {
	pendingForEpoch, ok := e.pendingState[epoch.Seq]
	if !ok {
		return
	}

	for _, party := range parties {
		pending, ok := pendingForEpoch[party]
		if !ok {
			continue
		}

		// get committed delegations for the party
		committedDelegations, ok := e.partyDelegationState[party]
		if !ok {
			committedDelegations = &partyDelegation{
				party:          party,
				totalDelegated: num.Zero(),
				nodeToAmount:   map[string]*num.Uint{},
			}
		}

		// apply undelegations deterministically
		nodeIDs := e.sortNodes(pending.nodeToUndelegateAmount)

		for _, nodeID := range nodeIDs {
			amount, ok := pending.nodeToUndelegateAmount[nodeID]
			if !ok {
				continue
			}
			committedForNode, delegationFoundForParty := committedDelegations.nodeToAmount[nodeID]
			if !delegationFoundForParty {
				// there is nothing to undelegate for this node, log and continue
				e.log.Debug("no committed delegation found for pending undelegation for", logging.String("party", party), logging.String("nodeID", nodeID))
				continue
			}

			validatorDelegation, ok := e.nodeDelegationState[nodeID]
			if !ok {
				// this should never happen
				e.log.Panic("trying to undelegate from an unknown node", logging.String("nodeID", nodeID))
			}

			validatorPartyDelegationAmount, ok := validatorDelegation.partyToAmount[party]
			if !ok == delegationFoundForParty {
				e.log.Panic("party and validator state disagree", logging.String("nodeID", nodeID), logging.String("party", party))
			}

			amountForUndelegate := amount
			if committedForNode.LT(amount) {
				amountForUndelegate = committedForNode
			}

			// undelegate
			// update validator mapping for the party
			validatorDelegation.partyToAmount[party] = num.Zero().Sub(validatorPartyDelegationAmount, amountForUndelegate)

			// if no more delegations for the party for the node, remove the mapping
			if validatorDelegation.partyToAmount[party].IsZero() {
				delete(validatorDelegation.partyToAmount, party)
			}
			validatorDelegation.totalDelegated = num.Zero().Sub(validatorDelegation.totalDelegated, amountForUndelegate)
			// if no more delegations for the node, clear it from the state
			if validatorDelegation.totalDelegated.IsZero() {
				delete(e.nodeDelegationState, nodeID)
			}

			// update undelegation for party
			committedDelegations.totalDelegated = num.Zero().Sub(committedDelegations.totalDelegated, amountForUndelegate)
			committedDelegations.nodeToAmount[nodeID] = num.Zero().Sub(committedForNode, amountForUndelegate)
			if committedDelegations.nodeToAmount[nodeID].IsZero() {
				delete(committedDelegations.nodeToAmount, nodeID)
			}

			if !committedDelegations.totalDelegated.IsZero() {
				e.partyDelegationState[party] = committedDelegations
			} else {
				_, ok := e.partyDelegationState[party]
				if ok {
					delete(e.partyDelegationState, party)
				}
			}
		}
	}
}

// process pending delegations for all parties
func (e *Engine) processPendingDelegations(parties []string, maxStakePerValidator *num.Uint, epoch types.Epoch) {
	pendingForEpoch, ok := e.pendingState[epoch.Seq]
	if !ok {
		return
	}

	// process undelegations for all parties first
	for _, party := range parties {
		pending, ok := pendingForEpoch[party]
		if !ok {
			continue
		}
		// get account balance
		partyBalance, err := e.stakingAccounts.GetAvailableBalance(party)
		if err != nil {
			e.log.Error("Failed to get available staking balance", logging.Error(err))
			continue
		}

		// get committed delegations for the party
		committedDelegations, ok := e.partyDelegationState[party]
		if !ok {
			committedDelegations = &partyDelegation{
				party:          party,
				totalDelegated: num.Zero(),
				nodeToAmount:   map[string]*num.Uint{},
			}
		}
		availableForDelegation := num.Zero().Sub(partyBalance, committedDelegations.totalDelegated)

		// apply delegation deterministically
		nodeIDs := e.sortNodes(pending.nodeToDelegateAmount)
		for _, nodeID := range nodeIDs {
			_, ok := pending.nodeToDelegateAmount[nodeID]
			if !ok {
				continue
			}

			// get the amount for delegation and adjust it if needed to the available balance for delegation in the validator
			amount := pending.nodeToDelegateAmount[nodeID].Clone()
			currentNodeDelegationBalance := num.Zero()
			currentNodeDelegation, ok := e.nodeDelegationState[nodeID]
			if ok {
				currentNodeDelegationBalance = currentNodeDelegation.totalDelegated
			}
			if !maxStakePerValidator.IsZero() {
				availableBalanceOnNode := num.Zero().Sub(maxStakePerValidator, currentNodeDelegationBalance)
				if amount.GT(availableBalanceOnNode) {
					amount = availableBalanceOnNode
				}
			}

			// check that the amount is not greater than the available for delegation
			if amount.GT(availableForDelegation) || amount.IsZero() {
				if e.log.GetLevel() <= logging.DebugLevel {
					e.log.Debug("the amount requested for delegation is greater than available for delegation at end of epoch", logging.String("party", party), logging.String("nodeID", nodeID), logging.BigUint("amt", amount))
				}
				continue
			}

			// update the validator delegation balance
			currentValidatorDelegation, ok := e.nodeDelegationState[nodeID]
			if !ok {
				currentValidatorDelegation = &validatorDelegation{
					nodeID:         nodeID,
					totalDelegated: num.Zero(),
					partyToAmount:  map[string]*num.Uint{},
				}
			}
			currentDelegationAmtForParty, ok := currentValidatorDelegation.partyToAmount[party]
			if !ok {
				currentDelegationAmtForParty = num.Zero()
			}
			currentValidatorDelegation.partyToAmount[party] = num.Sum(currentDelegationAmtForParty, amount)
			currentValidatorDelegation.totalDelegated = num.Sum(currentValidatorDelegation.totalDelegated, amount)
			e.nodeDelegationState[nodeID] = currentValidatorDelegation

			// update undelegation for party
			committedForNode, ok := committedDelegations.nodeToAmount[nodeID]
			if !ok {
				committedForNode = num.Zero()
			}
			committedDelegations.totalDelegated = num.Sum(committedDelegations.totalDelegated, amount)
			committedDelegations.nodeToAmount[nodeID] = num.Sum(committedForNode, amount)
			e.partyDelegationState[party] = committedDelegations
		}
	}
}

//returns the current state of the delegation per node
func (e *Engine) getValidatorData() []*types.ValidatorData {
	validatorNodes := e.topology.AllPubKeys()

	validators := make([]*types.ValidatorData, 0, len(validatorNodes))

	// sort the parties for deterministic handling
	sort.Strings(validatorNodes)

	for _, nodeID := range validatorNodes {
		validatorState, ok := e.nodeDelegationState[nodeID]
		if ok {
			validator := &types.ValidatorData{
				NodeID:     nodeID,
				Delegators: map[string]*num.Uint{},
			}
			selfStake := num.Zero()
			for delegatingParties, amt := range validatorState.partyToAmount {
				if delegatingParties == nodeID {
					selfStake = amt.Clone()
				} else {
					validator.Delegators[delegatingParties] = amt.Clone()
				}
			}
			validator.SelfStake = selfStake
			validator.StakeByDelegators = num.Zero().Sub(validatorState.totalDelegated, selfStake)
			validators = append(validators, validator)
			continue
		}
		// validator with no delegation at all
		validators = append(validators, &types.ValidatorData{
			NodeID:            nodeID,
			Delegators:        map[string]*num.Uint{},
			SelfStake:         num.Zero(),
			StakeByDelegators: num.Zero(),
		})
	}

	return validators

}
