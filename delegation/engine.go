package delegation

import (
	"context"
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var minRatioForAutoDelegation, _ = num.DecimalFromString("0.95")

const reconciliationInterval = 30 * time.Second

var (
	activeKey  = (&types.PayloadDelegationActive{}).Key()
	pendingKey = (&types.PayloadDelegationPending{}).Key()
	autoKey    = (&types.PayloadDelegationAuto{}).Key()
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

//TimeService notifies the reward engine on time updates
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/rewards TimeService
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
	GetTimeNow() time.Time
}

// ValidatorTopology represents the topology of validators and can check if a given node is a validator.
type ValidatorTopology interface {
	IsValidatorNode(nodeID string) bool
	AllNodeIDs() []string
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
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

// validator delegation state - updated at the end of each epoch.
type validatorDelegation struct {
	nodeID         string               // node id
	partyToAmount  map[string]*num.Uint // party -> delegated amount
	totalDelegated *num.Uint            // the total amount delegates by parties
}

// party delegation state - how much is delegated by the party to each validator and in total.
type partyDelegation struct {
	party          string               // party ID
	nodeToAmount   map[string]*num.Uint // nodeID -> delegated amount
	totalDelegated *num.Uint            // total amount delegated by party
}

// party delegation state.
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
// 2.3 process all pending delegations.
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
	compLevel            num.Decimal                                   // competition level
	minVal               num.Decimal                                   // minimum number of validators

	autoDelegationMode map[string]struct{} // parties entered auto-delegation mode
	dss                *delegationSnapshotState
	keyToSerialiser    map[string]func() ([]byte, error)
	lastReconciliation time.Time
}

// New instantiate a new delegation engine.
func New(log *logging.Logger, config Config, broker Broker, topology ValidatorTopology, stakingAccounts StakingAccounts, epochEngine EpochEngine, ts TimeService) *Engine {
	e := &Engine{
		config:               config,
		log:                  log.Named(namedLogger),
		broker:               broker,
		topology:             topology,
		stakingAccounts:      stakingAccounts,
		nodeDelegationState:  map[string]*validatorDelegation{},
		partyDelegationState: map[string]*partyDelegation{},
		pendingState:         map[uint64]map[string]*pendingPartyDelegation{},
		autoDelegationMode:   map[string]struct{}{},
		dss: &delegationSnapshotState{
			changed:    map[string]bool{activeKey: true, pendingKey: true, autoKey: true},
			hash:       map[string][]byte{},
			serialised: map[string][]byte{},
		},
		keyToSerialiser:    map[string]func() ([]byte, error){},
		lastReconciliation: time.Time{},
	}

	e.keyToSerialiser[activeKey] = e.serialiseActive
	e.keyToSerialiser[pendingKey] = e.serialisePending
	e.keyToSerialiser[autoKey] = e.serialiseAuto

	// register for epoch notifications
	epochEngine.NotifyOnEpoch(e.onEpochEvent)

	// register for time tick updates
	ts.NotifyOnTick(e.onChainTimeUpdate)

	return e
}

func (e *Engine) onChainTimeUpdate(ctx context.Context, t time.Time) {
	// if we've already done reconciliation (i.e. not first epoch) and it's been over <reconciliationIntervalSeconds> since, then reconcile.
	if (e.lastReconciliation != time.Time{}) && t.Sub(e.lastReconciliation) >= reconciliationInterval {
		// always reconcile the balance from the start of the epoch to the current time for simplicity
		e.reconcileAssociationWithNomination(ctx, e.currentEpoch.StartTime, t, e.currentEpoch.Seq)
	}
}

func (e *Engine) Hash() []byte {
	buf, err := e.Checkpoint()
	if err != nil {
		e.log.Panic("could not create checkpoint", logging.Error(err))
	}
	return crypto.Hash(buf)
}

// OnMinValidatorsChanged updates the network parameter for minValidators.
func (e *Engine) OnMinValidatorsChanged(ctx context.Context, minValidators int64) error {
	e.minVal = num.DecimalFromInt64(minValidators)
	return nil
}

// OnCompLevelChanged updates the network parameter for competitionLevel.
func (e *Engine) OnCompLevelChanged(ctx context.Context, compLevel float64) error {
	e.compLevel = num.DecimalFromFloat(compLevel)
	return nil
}

// OnMinAmountChanged updates the network parameter for minDelegationAmount.
func (e *Engine) OnMinAmountChanged(ctx context.Context, minAmount num.Decimal) error {
	e.minDelegationAmount, _ = num.UintFromDecimal(minAmount)
	return nil
}

// update the current epoch at which current pending delegations are recorded
// regardless if the event is start or stop of the epoch. the sequence is what identifies the epoch.
func (e *Engine) onEpochEvent(ctx context.Context, epoch types.Epoch) {
	if (e.lastReconciliation == time.Time{}) {
		e.lastReconciliation = epoch.StartTime
	}
	// if new epoch is starting we want to emit event for the next epoch for all delegations - this is because unless there is some action during the epoch
	// we will not emit an event for the next epoch until it starts - this will be more UI friendly
	if e.currentEpoch.Seq != epoch.Seq {
		// new epoch started - emit event for the next epoch
		parties := make([]string, 0, len(e.partyDelegationState))
		for p := range e.partyDelegationState {
			parties = append(parties, p)
		}
		sort.Strings(parties)
		for _, p := range parties {
			nodesSlice := make([]string, 0, len(e.partyDelegationState[p].nodeToAmount))
			for n := range e.partyDelegationState[p].nodeToAmount {
				nodesSlice = append(nodesSlice, n)
			}
			sort.Strings(nodesSlice)
			for _, n := range nodesSlice {
				e.sendNextEpochBalanceEvent(ctx, p, n, epoch.Seq)
			}
		}
	}
	e.currentEpoch = epoch
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

	// before we process pending we want to calculate how much is available for auto delegation
	partyToAutoDelegation, totalAvailableForAutoDelegation := e.eligiblePartiesForAutoDelegtion(epoch)
	// calculate the total number of tokens (a rough estimate) - this includes total delegated + applied pending + potential for auto delegation
	totalTokens := e.calcTotalDelegatedTokens(epoch.Seq, totalAvailableForAutoDelegation)
	// calculate the max for the next epoch
	numVal := len(e.topology.AllNodeIDs())
	maxStakePerValidator := e.calcMaxDelegatableTokens(totalTokens, num.DecimalFromInt64(int64(numVal)))
	// process pending undelegations/delegations
	e.processPending(ctx, epoch, maxStakePerValidator)
	// process auto delegations
	e.processAutoDelegation(partyToAutoDelegation, maxStakePerValidator)

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

	for p, state := range e.partyDelegationState {
		if _, ok := e.autoDelegationMode[p]; !ok {
			if balance, err := e.stakingAccounts.GetAvailableBalance(p); err == nil {
				if state.totalDelegated.ToDecimal().Div(balance.ToDecimal()).GreaterThanOrEqual(minRatioForAutoDelegation) {
					e.autoDelegationMode[p] = struct{}{}
					e.dss.changed[autoKey] = true
				}
			}
		}
	}

	// once in an epoch set changed to true
	e.dss.changed[activeKey] = true
	e.dss.changed[pendingKey] = true
	return stateForRewards
}

// Delegate increases the pending delegation balance and potentially decreases the pending undelegation balance for a given validator node.
func (e *Engine) Delegate(ctx context.Context, party string, nodeID string, amount *num.Uint) error {
	amt := amount.Clone()

	// check if the node is a validator node
	if !e.topology.IsValidatorNode(nodeID) {
		e.log.Error("Trying to delegate to an invalid node", logging.String("party", party), logging.String("nodeID", nodeID))
		return ErrInvalidNodeID
	}

	// check if the delegator has a staking account
	partyBalance, err := e.stakingAccounts.GetAvailableBalance(party)
	if err != nil {
		e.log.Error("Party has no staking account balance", logging.String("party", party), logging.String("nodeID", nodeID))
		return ErrPartyHasNoStakingAccount
	}

	if amt.LT(e.minDelegationAmount) {
		e.log.Error("Amount for delegation is lower than minimum required amount", logging.String("party", party), logging.String("nodeID", nodeID), logging.String("amount", num.UintToString(amount)), logging.String("amount", num.UintToString(e.minDelegationAmount)))
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
		e.log.Error("Party has insufficient account balance", logging.String("party", party), logging.String("nodeID", nodeID), logging.String("partyBalance", num.UintToString(partyBalance)), logging.String("partyDelegationBalance", num.UintToString(partyDelegationBalance)), logging.String("amount", num.UintToString(amount)))
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
			e.log.Error("Party has insufficient account balance", logging.String("party", party), logging.String("nodeID", nodeID), logging.String("partyBalance", num.UintToString(partyBalance)), logging.String("balanceAvailableForDelegation", num.UintToString(balanceAvailableForDelegation)), logging.String("partyPendingDelegation", num.UintToString(partyPendingDelegation)), logging.String("amount", num.UintToString(amount)))
			return ErrInsufficientBalanceForDelegation
		}
		balanceAvailableForDelegation = num.Zero().Sub(balanceAvailableForDelegation, partyPendingDelegation)
	}

	// if the balance with committed and pending delegations/undelegations is insufficient to satisfy the delegation return error
	if balanceAvailableForDelegation.LT(amt) {
		e.log.Error("Party has insufficient account balance", logging.String("party", party), logging.String("nodeID", nodeID), logging.String("partyBalance", num.UintToString(partyBalance)), logging.String("balanceAvailableForDelegation", num.UintToString(balanceAvailableForDelegation)), logging.String("amount", num.UintToString(amount)))
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
	e.dss.changed[pendingKey] = true
	return nil
}

// UndelegateAtEndOfEpoch increases the pending undelegation balance and potentially decreases the pending delegation balance for a given validator node and party.
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
		e.log.Error("Trying to delegate to an invalid node", logging.String("party", party), logging.String("nodeID", nodeID))
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
		e.log.Error("Invalid undelegation - trying to undelegate more than delegated", logging.String("party", party), logging.String("nodeID", nodeID), logging.String("undelegationAmount", num.UintToString(amt)), logging.String("totalDelegationBalance", num.UintToString(totalDelegationBalance)))
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
	e.dss.changed[pendingKey] = true
	return nil
}

// UndelegateNow changes the balance of delegation immediately without waiting for the end of the epoch
// if possible it removed balance from pending delegated, if not enough it removes balance from the current epoch delegated amount.
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
	amt := amount.Clone()
	if amt.IsZero() {
		amt = totalAvailableForUndelegation.Clone()
	}

	if amt.GT(totalAvailableForUndelegation) {
		e.log.Error("Invalid undelegation - trying to undelegate more than delegated", logging.String("party", party), logging.String("nodeID", nodeID), logging.String("amt", num.UintToString(amt)), logging.String("totalAvailableForUndelegation", num.UintToString(totalAvailableForUndelegation)))
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

	// get out of auto delegation mode
	delete(e.autoDelegationMode, party)
	e.dss.changed[autoKey] = true
	e.dss.changed[activeKey] = true
	e.dss.changed[pendingKey] = true
	return nil
}

func (e *Engine) getNextEpochBalanceEvent(ctx context.Context, party, nodeID string, seq uint64) events.Event {
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

	potentialDelegationForNextEpoch := num.Sum(delegatedToNode, pendingDelegated)
	amt := num.Zero()
	if potentialDelegationForNextEpoch.GT(pendingUndelegated) {
		amt = num.Zero().Sub(potentialDelegationForNextEpoch, pendingUndelegated)
	}
	return events.NewDelegationBalance(ctx, party, nodeID, amt, num.NewUint(seq+1).String())
}

// sends the expected balance for the next epoch.
func (e *Engine) sendNextEpochBalanceEvent(ctx context.Context, party, nodeID string, seq uint64) {
	e.broker.Send(e.getNextEpochBalanceEvent(ctx, party, nodeID, seq))
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

// reconcileAssociationWithNomination makes sure that current epoch's nomination has sufficient cover in association and if not
// adjusts the nomination accordingly.
func (e *Engine) reconcileAssociationWithNomination(ctx context.Context, from, to time.Time, epochSeq uint64) {
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
		stakeBalance, err := e.stakingAccounts.GetAvailableBalanceInRange(party, from, to)
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
			e.sendDelegatedBalanceEvent(ctx, party, nodeID, epochSeq)
			e.sendNextEpochBalanceEvent(ctx, party, nodeID, epochSeq)
		}

		// get out of auto delegation mode
		delete(e.autoDelegationMode, party)
	}
	e.lastReconciliation = to
}

// preprocessEpoch is called at the end of an epoch and updates the state to be returned for rewarding calculation
// check balance for the epoch duration and undelegate if delegations don't have sufficient cover
// the state of the engine by the end of this method reflects the state to be used for reward engine.
func (e *Engine) preprocessEpochForRewarding(ctx context.Context, epoch types.Epoch) {
	e.reconcileAssociationWithNomination(ctx, epoch.StartTime, epoch.EndTime, epoch.Seq)
}

// calculate the total number of tokens (a rough estimate) and the number of nodes.
func (e *Engine) calcTotalDelegatedTokens(epochSeq uint64, availableForAutoDelegation *num.Uint) *num.Uint {
	totalDelegatedTokens := num.Zero()
	for _, nodeDel := range e.nodeDelegationState {
		totalDelegatedTokens.AddSum(nodeDel.totalDelegated)
	}
	if pendingForEpoch, ok := e.pendingState[epochSeq]; ok {
		for _, pendingDel := range pendingForEpoch {
			totalDelegatedTokens = totalDelegatedTokens.Sub(totalDelegatedTokens.AddSum(pendingDel.totalDelegation), pendingDel.totalUndelegation)
		}
	}
	// include auto delegation
	totalDelegatedTokens.AddSum(availableForAutoDelegation)
	return totalDelegatedTokens
}

func (e *Engine) calcMaxDelegatableTokens(totalTokens *num.Uint, numVal num.Decimal) *num.Uint {
	a := num.MaxD(e.minVal, numVal.Div(e.compLevel))

	res, _ := num.UintFromDecimal(totalTokens.ToDecimal().Div(a))
	return res
}

// process pending delegations and undelegations at the end of the epoch and clear the delegation/undelegation maps at the end.
func (e *Engine) processPending(ctx context.Context, epoch types.Epoch, maxStakePerValidator *num.Uint) {
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

	// read the delegation min amount network param
	e.processPendingUndelegations(parties, epoch)
	e.processPendingDelegations(parties, maxStakePerValidator, epoch)

	delete(e.pendingState, epoch.Seq)
}

// process pending undelegations for all parties.
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

		if pending.totalUndelegation.IsZero() {
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
		// undelegation removes the party from auto delegation mode
		delete(e.autoDelegationMode, party)
	}
}

// process pending delegations for all parties.
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

		// partyBalance must be >= totalDelegated as we've already adjusted the balance before this is called, if not just log and carry on
		if partyBalance.LT(committedDelegations.totalDelegated) {
			e.log.Warn("Party balance is less than delegated balance when processing pending delegations",
				logging.Uint64("epoch", epoch.Seq),
				logging.String("party", party),
				logging.String("associationBalance", partyBalance.String()),
				logging.String("delegationBalance", committedDelegations.totalDelegated.String()),
			)
			continue
		}

		// this is how much is left associated unnominated
		availableForDelegation := num.Zero().Sub(partyBalance, committedDelegations.totalDelegated)

		// if there's no balance left, nothing to do here
		if availableForDelegation.IsZero() {
			continue
		}

		// apply delegation deterministically
		nodeIDs := e.sortNodes(pending.nodeToDelegateAmount)
		nodeIDToExpectedAmount := make([]*num.Uint, 0, len(nodeIDs))
		totalExpectedDelegation := num.Zero()

		// first calculate how much we're expecting to delegate to each node and the total delegation amount
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

			// record the relevant amount and add it to the total
			nodeIDToExpectedAmount = append(nodeIDToExpectedAmount, amount)
			totalExpectedDelegation.AddSum(amount)

			if amount.IsZero() {
				if e.log.GetLevel() <= logging.DebugLevel {
					e.log.Debug("the amount requested for delegation is greater than available for delegation at end of epoch", logging.String("party", party), logging.String("nodeID", nodeID), logging.BigUint("amt", amount))
				}
				continue
			}
		}

		// if we don't have enough to satisfy the delegation amounts - prorate them with respect to how much is available
		if totalExpectedDelegation.GT(availableForDelegation) {
			for i, amt := range nodeIDToExpectedAmount {
				factor := amt.ToDecimal().Div(totalExpectedDelegation.ToDecimal())
				nodeIDToExpectedAmount[i], _ = num.UintFromDecimal(availableForDelegation.ToDecimal().Mul(factor))
			}
		}

		// no apply the delegations with the prorated amount
		for i, nodeID := range nodeIDs {
			amount := nodeIDToExpectedAmount[i]
			if amount.IsZero() {
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

// eligiblePartiesForAutoDelegtion calculates how much is available for auto delegation in parties that have qualifies for auto delegation
// and have not done any manual actions during the past epoch and have any active delegations and have available balance.
func (e *Engine) eligiblePartiesForAutoDelegtion(epoch types.Epoch) (map[string]*num.Uint, *num.Uint) {
	totalAvailableForAutoDelegation := num.Zero()
	partyToAvailableBalance := map[string]*num.Uint{}
	for party := range e.autoDelegationMode {
		// if the party didn't attempt to do any manual delegations during the epoch and they have any undelegated balance we capture this balance for auto delegation
		if epochPending, ok := e.pendingState[epoch.Seq]; ok {
			if _, ok = epochPending[party]; ok {
				continue
			}
		}

		// if the party has no delegation we can't auto delegate
		if _, ok := e.partyDelegationState[party]; !ok {
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
		available := num.Zero().Sub(balance, delegated)
		if !available.IsZero() {
			partyToAvailableBalance[party] = available
		}
		totalAvailableForAutoDelegation.AddSum(available)
	}
	return partyToAvailableBalance, totalAvailableForAutoDelegation
}

// processAutoDelegation takes a slice of parties which are known to be eligible for auto delegation and attempts to distribute their available
// undelegated stake proportionally across the nodes to which it already delegated to.
// It respects the max delegation per validator, and if the node does not accept any more stake it will not try to delegate it to other nodes.
func (e *Engine) processAutoDelegation(partyToAvailableBalance map[string]*num.Uint, maxPerNode *num.Uint) {
	parties := make([]string, 0, len(partyToAvailableBalance))
	for p := range partyToAvailableBalance {
		parties = append(parties, p)
	}
	sort.Strings(parties)

	for _, p := range parties {
		totalDelegation := e.partyDelegationState[p].totalDelegated.ToDecimal()
		balanceDec := partyToAvailableBalance[p].ToDecimal()
		for n, nodeBalance := range e.partyDelegationState[p].nodeToAmount {
			ratio := nodeBalance.ToDecimal().Div(totalDelegation)
			delegationToNodeN, _ := num.UintFromDecimal(ratio.Mul(balanceDec))
			if e.nodeDelegationState[n].totalDelegated.GTE(maxPerNode) {
				continue
			}
			spaceLeftOnN := num.Zero().Sub(maxPerNode, e.nodeDelegationState[n].totalDelegated)
			delegationToNodeN = num.Min(delegationToNodeN, spaceLeftOnN)
			if !delegationToNodeN.IsZero() {
				e.partyDelegationState[p].totalDelegated.AddSum(delegationToNodeN)
				e.partyDelegationState[p].nodeToAmount[n].AddSum(delegationToNodeN)
				e.nodeDelegationState[n].totalDelegated.AddSum(delegationToNodeN)
				e.nodeDelegationState[n].partyToAmount[p].AddSum(delegationToNodeN)
			}
		}
	}
}

// returns the current state of the delegation per node.
func (e *Engine) getValidatorData() []*types.ValidatorData {
	validatorNodes := e.topology.AllNodeIDs()

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
