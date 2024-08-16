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

package statevar

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

// ConsensusState trakcs the state transitions of a state variable.
type ConsensusState int

const (
	ConsensusStateUnspecified ConsensusState = iota
	ConsensusStateCalculationStarted
	ConsensusStatePerfectMatch
	ConsensusStateSeekingConsensus
	ConsensusStateconsensusReachedLocked
	ConsensusStateCalculationAborted
	ConsensusStateError
	ConsensusStateStale
)

var stateToName = map[ConsensusState]string{
	ConsensusStateUnspecified:            "undefined",
	ConsensusStateCalculationStarted:     "consensus_calc_started",
	ConsensusStatePerfectMatch:           "perfect_match",
	ConsensusStateSeekingConsensus:       "seeking_consensus",
	ConsensusStateconsensusReachedLocked: "consensus_reached",
	ConsensusStateCalculationAborted:     "consensus_calc_aborted",
	ConsensusStateError:                  "error",
}

type StateVariable struct {
	log              *logging.Logger
	top              Topology
	cmd              Commander
	broker           Broker
	ID               string                                                    // the unique identifier of the state variable
	asset            string                                                    // the asset of the state variable - used for filtering relevant events
	market           string                                                    // the market of the state variable - used for filtering relevant events
	converter        statevar.Converter                                        // convert to/from the key/value bundle model into typed result model
	startCalculation func(string, statevar.FinaliseCalculation)                // a callback to the owner to start the calculation of the value of the state variable
	result           func(context.Context, statevar.StateVariableResult) error // a callback to be called when the value reaches consensus

	state                       ConsensusState                      // the current status of consensus
	eventID                     string                              // the event ID triggering the calculation
	validatorResults            map[string]*statevar.KeyValueBundle // the result of the calculation as received from validators
	roundsSinceMeaningfulUpdate uint
	pendingEvents               []pendingEvent
	lock                        sync.Mutex

	currentTime time.Time

	// use retries to workaround transactions go missing in tendermint
	lastSentSelfBundle     *commandspb.StateVariableProposal
	lastSentSelfBundleTime time.Time
}

func NewStateVar(
	log *logging.Logger,
	broker Broker,
	top Topology,
	cmd Commander,
	currentTime time.Time,
	ID, asset,
	market string,
	converter statevar.Converter,
	startCalculation func(string, statevar.FinaliseCalculation),
	trigger []statevar.EventType,
	result func(context.Context, statevar.StateVariableResult) error,
) *StateVariable {
	sv := &StateVariable{
		log:                         log,
		broker:                      broker,
		top:                         top,
		cmd:                         cmd,
		ID:                          ID,
		asset:                       asset,
		market:                      market,
		converter:                   converter,
		startCalculation:            startCalculation,
		result:                      result,
		state:                       ConsensusStateUnspecified,
		validatorResults:            map[string]*statevar.KeyValueBundle{},
		roundsSinceMeaningfulUpdate: 0,
	}
	return sv
}

// GetAsset returns the asset of the state variable.
func (sv *StateVariable) GetAsset() string {
	return sv.asset
}

// GetMarket returns the market of the state variable.
func (sv *StateVariable) GetMarket() string {
	return sv.market
}

// endBlock is called at the end of the block to flush the event. This is snapshot-friendly so that at the end of the block we clear all events as opposed to doing the same at the beginning of the block.
func (sv *StateVariable) endBlock(ctx context.Context) {
	sv.lock.Lock()
	evts := make([]events.Event, 0, len(sv.pendingEvents))
	for _, pending := range sv.pendingEvents {
		newEvt := events.NewStateVarEvent(ctx, sv.ID, pending.eventID, pending.state)
		evts = append(evts, newEvt)
		protoEvt := newEvt.Proto()
		if sv.log.IsDebug() {
			sv.log.Debug("state-var event sent", logging.String("event", protoEvt.String()))
		}
	}
	sv.pendingEvents = []pendingEvent{}
	sv.lock.Unlock()
	sv.broker.SendBatch(evts)
}

func (sv *StateVariable) startBlock(t time.Time) {
	sv.lock.Lock()
	sv.currentTime = t

	// if we have an active event, and we sent the bundle and we're 5 seconds after sending the bundle and haven't received our self bundle
	// that means the transaction may have gone missing, let's retry sending it.
	needsResend := false
	if sv.eventID != "" && sv.lastSentSelfBundle != nil && t.After(sv.lastSentSelfBundleTime.Add(5*time.Second)) {
		sv.lastSentSelfBundleTime = t
		needsResend = true
	}
	sv.lock.Unlock()
	if needsResend {
		sv.logAndRetry(errors.New("consensus not reached - timeout expired"), sv.lastSentSelfBundle)
	}
}

// calculation is required for the state variable for the given event id.
func (sv *StateVariable) eventTriggered(eventID string) {
	sv.lock.Lock()

	if sv.log.IsDebug() {
		sv.log.Debug("event triggered", logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}
	// if we get a new event while processing an existing event we abort the current calculation and start a new one
	if sv.eventID != "" {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("aborting state variable event", logging.String("state-var", sv.ID), logging.String("aborted-event-id", sv.eventID), logging.String("new-event-id", sv.eventID))
		}

		// reset the last bundle so we don't send it by mistake
		sv.lastSentSelfBundle = nil

		// if we got a new event and were not in consensus, increase the number of rounds with no consensus and if
		// we've not had a meaningful update - send an event with stale state
		if sv.state == ConsensusStateSeekingConsensus {
			sv.roundsSinceMeaningfulUpdate++
			if sv.roundsSinceMeaningfulUpdate >= 3 {
				sv.state = ConsensusStateStale
				sv.addEventLocked()
			}
		}

		sv.state = ConsensusStateCalculationAborted
		sv.addEventLocked()
	}

	// reset any existing state
	sv.eventID = eventID
	sv.validatorResults = map[string]*statevar.KeyValueBundle{}
	sv.state = ConsensusStateCalculationStarted
	sv.addEventLocked()

	sv.lock.Unlock()

	// kickoff calculation
	sv.startCalculation(sv.eventID, sv)
}

// CalculationFinished is called from the owner when the calculation is completed to kick off consensus.
func (sv *StateVariable) CalculationFinished(eventID string, result statevar.StateVariableResult, err error) {
	sv.lock.Lock()
	if sv.eventID != eventID {
		sv.log.Warn("ignoring recevied the result of a calculation of an old eventID", logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}
	if err != nil {
		sv.log.Error("could not calculate state for", logging.String("id", sv.ID), logging.String("event-id", eventID))
		sv.state = ConsensusStateError
		sv.addEventLocked()
		sv.eventID = ""
		sv.lock.Unlock()
		return
	}

	if !sv.top.IsValidator() {
		// if we're a non-validator we still need to do the calculation so that the snapshot will be in sync with
		// a validators, but now we're here we do not need to actually send in our results.
		sv.lock.Unlock()
		return
	}

	// save our result and send the result to vega to be updated by other nodes.
	kvb := sv.converter.InterfaceToBundle(result).ToProto()

	// this is a test feature that adds noise up to the tolerance to the state variable
	// it should be excluded by build tag for production
	kvb = sv.AddNoise(kvb)

	svp := &commandspb.StateVariableProposal{
		Proposal: &vegapb.StateValueProposal{
			StateVarId: sv.ID,
			EventId:    sv.eventID,
			Kvb:        kvb,
		},
	}

	// set the bundle and the time
	sv.lastSentSelfBundle = svp
	sv.lastSentSelfBundleTime = sv.currentTime

	// need to release the lock before we send the transaction command
	sv.lock.Unlock()
	sv.cmd.Command(context.Background(), txn.StateVariableProposalCommand, svp, func(_ string, err error) { sv.logAndRetry(err, svp) }, nil)
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("result calculated and sent to vega", logging.String("validator", sv.top.SelfNodeID()), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}
}

// logAndRetry logs errors from tendermint transaction submission failure and retries if we're still handling the same event.
func (sv *StateVariable) logAndRetry(err error, svp *commandspb.StateVariableProposal) {
	if err == nil {
		return
	}
	sv.lock.Lock()
	//	sv.log.Error("failed to send state variable proposal command", logging.String("id", sv.ID), logging.String("event-id", sv.eventID), logging.Error(err))
	if svp.Proposal.EventId == sv.eventID {
		sv.lock.Unlock()
		if sv.log.IsDebug() {
			sv.log.Debug("retrying to send state variable proposal command", logging.String("id", sv.ID), logging.String("event-id", sv.eventID))
		}
		sv.cmd.Command(context.Background(), txn.StateVariableProposalCommand, svp, func(_ string, err error) { sv.logAndRetry(err, svp) }, nil)
		return
	}
	sv.lock.Unlock()
}

// bundleReceived is called when we get a result from another validator corresponding to a given event ID.
func (sv *StateVariable) bundleReceived(ctx context.Context, node, eventID string, bundle *statevar.KeyValueBundle, rng *rand.Rand, validatorVotesRequired num.Decimal) {
	sv.lock.Lock()
	defer sv.lock.Unlock()

	// if the bundle is received for a stale or wrong event, ignore it
	if sv.eventID != eventID {
		sv.log.Debug("received a result for a stale event", logging.String("ID", sv.ID), logging.String("from-node", node), logging.String("current-even-id", sv.eventID), logging.String("receivedEventID", eventID))
		return
	}

	// if for some reason we received a result from a non validator node, ignore it
	if !sv.top.IsValidatorVegaPubKey(node) {
		sv.log.Debug("state var bundle received from a non validator node - ignoring", logging.String("from-validator", node), logging.String("state-var", sv.ID), logging.String("eventID", eventID))
		return
	}

	if sv.top.SelfNodeID() == node {
		sv.lastSentSelfBundle = nil
		sv.lastSentSelfBundleTime = time.Time{}
		sv.log.Debug("state var bundle received self vote", logging.String("from-validator", node), logging.String("state-var", sv.ID), logging.String("eventID", eventID))
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var bundle received", logging.String("from-validator", node), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}

	if sv.state == ConsensusStatePerfectMatch || sv.state == ConsensusStateconsensusReachedLocked {
		sv.log.Debug("state var bundle received, consensus already reached", logging.String("from-validator", node), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
		return
	}

	// save the result from the validator and check if we have a quorum
	sv.validatorResults[node] = bundle

	// calculate how much voting power is required for majority
	requiredVotingPower := validatorVotesRequired.Mul(num.DecimalFromInt64(sv.top.GetTotalVotingPower()))

	// calculate how much voting power is represented by the voters
	bundlesVotingPower := num.DecimalZero()
	for k := range sv.validatorResults {
		bundlesVotingPower = bundlesVotingPower.Add(num.DecimalFromInt64(sv.top.GetVotingPower(k)))
	}

	if sv.log.IsDebug() {
		sv.log.Debug("received results for state variable", logging.String("state-var", sv.ID), logging.String("event-id", eventID), logging.Decimal("received-voting-power", bundlesVotingPower), logging.String("out-of", requiredVotingPower.String()))
	}

	if bundlesVotingPower.LessThan(requiredVotingPower) {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("waiting for more results for state variable consensus check", logging.String("state-var", sv.ID), logging.Decimal("received-voting-power", bundlesVotingPower), logging.String("out-of", requiredVotingPower.String()))
		}
		return
	}

	// if we're already in seeking consensus state, no point in checking if all match - suffice checking if there's a majority with matching within tolerance
	if sv.state == ConsensusStateSeekingConsensus {
		sv.tryConsensusLocked(ctx, rng, requiredVotingPower)
		return
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var checking consensus (2/3 of the results received", logging.String("from-validator", node), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}

	// we got enough results lets check if they match
	var result *statevar.KeyValueBundle
	// check if results from all validator totally agree
	for nodeID, res := range sv.validatorResults {
		if result == nil {
			result = res
		}
		if !sv.validatorResults[nodeID].Equals(result) {
			if sv.log.GetLevel() <= logging.DebugLevel {
				sv.log.Debug("state var consensus NOT reached through perfect match", logging.String("state-var", sv.ID), logging.String("event-id", eventID), logging.Int("num-results", len(sv.validatorResults)))
			}

			// initiate a round of voting
			sv.state = ConsensusStateSeekingConsensus
			sv.tryConsensusLocked(ctx, rng, requiredVotingPower)
			return
		}
	}

	// we are done - happy days!
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var consensus reached through perfect match", logging.String("state-var", sv.ID), logging.String("event-id", eventID), logging.Int("num-results", len(sv.validatorResults)))
	}
	sv.state = ConsensusStatePerfectMatch
	// convert the result to decimal and let the owner of the state variable know
	sv.consensusReachedLocked(ctx, result)
}

// if the bundles are not all equal to each other, choose one at random and verify that all others are within tolerance.
// NB: assumes lock has already been acquired.
func (sv *StateVariable) tryConsensusLocked(ctx context.Context, rng *rand.Rand, requiredVotingPower num.Decimal) {
	// sort the node IDs for determinism
	nodeIDs := make([]string, 0, len(sv.validatorResults))
	for nodeID := range sv.validatorResults {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	alreadyCheckedForTolerance := map[string]struct{}{}

	for len(alreadyCheckedForTolerance) != len(nodeIDs) {
		nodeID := nodeIDs[rng.Intn(len(nodeIDs))]
		if _, ok := alreadyCheckedForTolerance[nodeID]; ok {
			continue
		}
		alreadyCheckedForTolerance[nodeID] = struct{}{}
		candidateResult := sv.validatorResults[nodeID]
		votingPowerMatch := num.DecimalZero()
		for _, nID := range nodeIDs {
			if sv.validatorResults[nID].WithinTolerance(candidateResult) {
				votingPowerMatch = votingPowerMatch.Add(num.DecimalFromInt64(sv.top.GetVotingPower(nID)))
			}
		}
		if votingPowerMatch.GreaterThanOrEqual(requiredVotingPower) {
			sv.state = ConsensusStateconsensusReachedLocked
			sv.consensusReachedLocked(ctx, candidateResult)
			return
		}
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var consensus NOT reached through random selection", logging.String("state-var", sv.ID), logging.String("event-id", sv.eventID), logging.Int("num-results", len(sv.validatorResults)))
	}
}

// consensus was reached either through a vote or through perfect matching of all of 2/3 of the validators.
// NB: assumes lock has already been acquired.
func (sv *StateVariable) consensusReachedLocked(ctx context.Context, acceptedValue *statevar.KeyValueBundle) {
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("consensus reached", logging.String("state-var", sv.ID), logging.String("event-id", sv.eventID))
	}

	sv.result(ctx, sv.converter.BundleToInterface(acceptedValue))
	sv.addEventLocked()

	if sv.log.IsDebug() {
		sv.log.Debug("consensus reached for state variable", logging.String("state-var", sv.ID), logging.String("event-id", sv.eventID))
	}

	// reset the state
	sv.eventID = ""
	sv.validatorResults = nil
	sv.roundsSinceMeaningfulUpdate = 0
}

// addEventLocked adds an event to the pending events.
// NB: assumes lock has already been acquired.
func (sv *StateVariable) addEventLocked() {
	sv.pendingEvents = append(sv.pendingEvents, pendingEvent{sv.eventID, stateToName[sv.state]})
}

type pendingEvent struct {
	eventID string
	state   string
}
