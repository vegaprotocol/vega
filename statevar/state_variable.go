package statevar

import (
	"context"
	"math/rand"
	"sort"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

// StateVarConsensusState trakcs the state transitions of a state variable.
type StateVarConsensusState int

const (
	StateVarConsensusStateUnspecified         StateVarConsensusState = iota
	StateVarConsensusStateCalculationStarted                         = iota
	StateVarConsensusStatePerfectMatch                               = iota
	StateVarConsensusStateConsensusReached                           = iota
	StateVarConsensusStateConsensusNotReached                        = iota
	StateVarConsensusStateCalculationAborted                         = iota
	StateVarConsensusStateError                                      = iota
)

var stateToName = map[StateVarConsensusState]string{
	StateVarConsensusStateUnspecified:         "undefined",
	StateVarConsensusStateCalculationStarted:  "consensus_calc_started",
	StateVarConsensusStatePerfectMatch:        "perfect_match",
	StateVarConsensusStateConsensusReached:    "consensus_reached",
	StateVarConsensusStateConsensusNotReached: "consensus_not_reached",
	StateVarConsensusStateCalculationAborted:  "consensus_calc_aborted",
	StateVarConsensusStateError:               "error",
}

type StateVariable struct {
	log    *logging.Logger
	top    Topology
	cmd    Commander
	broker Broker
	// configuration
	ID            string                                   // the unique identifier of the state variable
	calculateFunc func() (*statevar.KeyValueBundle, error) // a callback to the owner to calculate the value of the state variable
	trigger       []StateVarEventType                      // events that should trigger the calculation of the state variable
	frequency     time.Duration                            // the frequency for time based triggering
	result        func(*statevar.KeyValueResult) error     // a callback to be called when the value reaches consensus

	// state
	nextTimeToRun    time.Time                           // the next scheduled calculation
	state            StateVarConsensusState              // the current status of consensus
	eventID          string                              // the event ID triggering the calculation
	validatorResults map[string]*statevar.KeyValueBundle // the result of the calculation as received from validators
	currentValue     *statevar.KeyValueResult            // the current result
}

func NewStateVar(log *logging.Logger, broker Broker, top Topology, cmd Commander, currentTime time.Time, ID string, calculateFunc func() (*statevar.KeyValueBundle, error), trigger []StateVarEventType, frequency time.Duration, result func(*statevar.KeyValueResult) error, defaultValue *statevar.KeyValueResult) *StateVariable {
	// if frequency is specified, "schedule" a calculation for now
	nextTimeToRun := time.Time{}
	if frequency != time.Duration(0) {
		nextTimeToRun = currentTime
	}
	sv := &StateVariable{
		log:              log,
		broker:           broker,
		top:              top,
		cmd:              cmd,
		ID:               ID,
		calculateFunc:    calculateFunc,
		trigger:          trigger,
		result:           result,
		state:            StateVarConsensusStateUnspecified,
		validatorResults: map[string]*statevar.KeyValueBundle{},
		nextTimeToRun:    nextTimeToRun,
		currentValue:     defaultValue,
	}
	sv.currentValue = &statevar.KeyValueResult{
		KeyDecimalValue: defaultValue.KeyDecimalValue,
		Validity:        statevar.StateValidityDefault,
	}
	result(sv.currentValue)
	return sv
}

// calculation is required for the state variable for the given event id.
func (sv *StateVariable) eventTriggered(eventID string) error {
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("eventTriggered", logging.String("stateVar", sv.ID), logging.String("eventID", eventID))
	}

	// if we get a new event while processing an existing event we abort the current calculation and start a new one
	if sv.eventID != "" {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("aborting state variable event", logging.String("stateVar", sv.ID), logging.String("abortedEventID", sv.eventID), logging.String("newEventID", sv.eventID))
		}
		sv.state = StateVarConsensusStateCalculationAborted
		sv.sendEvent()
	}

	// reset any existing state
	sv.eventID = eventID
	sv.validatorResults = map[string]*statevar.KeyValueBundle{}
	sv.state = StateVarConsensusStateCalculationStarted

	// if we are a validator we save our result
	if sv.top.IsValidator() {
		// request calculation of the state from the owner of the component
		candidateState, err := sv.calculateFunc()
		if err != nil {
			sv.log.Error("could not calculate state for", logging.String("ID", sv.ID), logging.String("eventID", eventID))
			sv.state = StateVarConsensusStateError
			sv.sendEvent()
			sv.eventID = ""
			return err
		}

		// save our result and send the result to vega to be updated by other nodes.
		sv.validatorResults[sv.top.SelfNodeID()] = candidateState
		svp := &commandspb.StateVariableProposal{}
		sv.cmd.Command(context.Background(), txn.StateVariableProposalCommand, svp, func(error) {})
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("result calculated and sent to vega", logging.String("validator", sv.top.SelfNodeID()), logging.String("stateVar", sv.ID), logging.String("eventID", eventID))
		}
	}
	return nil
}

// bundleReceived is called when we get a result from another validator corresponding to a given event ID.
func (sv *StateVariable) bundleReceived(nodeID, eventID string, bundle *statevar.KeyValueBundle, rng *rand.Rand, validatorVotesRequired num.Decimal) {
	// if the bundle is received for a stale or wrong event, ignore it
	if sv.eventID != eventID {
		sv.log.Debug("received a result for a stale event", logging.String("ID", sv.ID), logging.String("fromNode", nodeID), logging.String("currentEventID", sv.eventID), logging.String("receivedEventID", eventID))
		return
	}

	// if for some reason we received a result from a non validator node, ignore it
	if !sv.top.IsValidatorNodeID(nodeID) {
		return
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var bundle received", logging.String("fromValidator", nodeID), logging.String("stateVar", sv.ID), logging.String("eventID", eventID))
	}

	// save the result from the validator and check if we have a quorum
	sv.validatorResults[nodeID] = bundle
	numResults := num.DecimalFromInt64(int64(len(sv.validatorResults)))
	validatorsNum := num.DecimalFromInt64(int64(len(sv.top.AllNodeIDs())))
	if !numResults.Div(validatorsNum).GreaterThanOrEqual(validatorVotesRequired) {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("waiting for more results for state variable consensus check", logging.String("stateVar", sv.ID), logging.String("eventID", eventID), logging.String("received", numResults.String()), logging.String("outOf", validatorsNum.String()))
		}
		return
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var checking consensus (2/3 of the results received", logging.String("fromValidator", nodeID), logging.String("stateVar", sv.ID), logging.String("eventID", eventID))
	}

	// we got enough results lets check if they match
	allMatch := true
	var result *statevar.KeyValueBundle
	// check if results from all validator totally agree
	for nodeID, res := range sv.validatorResults {
		if result == nil {
			result = res
		}
		allMatch = allMatch && sv.validatorResults[nodeID].Equals(result)
	}

	if !allMatch {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("state var consensus NOT reached through perfect match", logging.String("stateVar", sv.ID), logging.String("eventID", eventID), logging.Int("numResults", len(sv.validatorResults)))
		}

		// initiate a round of voting
		sv.tryConsensus(rng, validatorVotesRequired)
		return
	}

	// we are done - happy days!
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var consensus reached through perfect match", logging.String("stateVar", sv.ID), logging.String("eventID", eventID), logging.Int("numResults", len(sv.validatorResults)))
	}
	sv.state = StateVarConsensusStatePerfectMatch
	// convert the result to decimal and let the owner of the state variable know
	sv.consensusReached(result.ToDecimal())
}

// if the bundles are not all equal to each other, choose one at random and verify that all others are within tolerance, if none can be found, mark the value as stale.
func (sv *StateVariable) tryConsensus(rng *rand.Rand, validatorVotesRequired num.Decimal) {
	// sort the node IDs for determinism
	nodeIDs := make([]string, 0, len(sv.validatorResults))
	for nodeID := range sv.validatorResults {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	alreadyUsed := map[string]struct{}{}
	for {
		if len(alreadyUsed) == len(nodeIDs) {
			break
		}
		nodeID := nodeIDs[rng.Intn(len(nodeIDs))]
		if _, ok := alreadyUsed[nodeID]; ok {
			continue
		}
		alreadyUsed[nodeID] = struct{}{}
		candidateResult := sv.validatorResults[nodeID]
		countMatch := int64(0)
		for _, res := range sv.validatorResults {
			if res.WithinTolerance(candidateResult) {
				countMatch++
			}
		}
		if num.DecimalFromInt64(countMatch).Div(num.DecimalFromInt64(int64(len(sv.validatorResults)))).GreaterThanOrEqual(validatorVotesRequired) {
			sv.consensusReached(candidateResult.ToDecimal())
			return
		}
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var consensus NOT reached through random selection", logging.String("stateVar", sv.ID), logging.String("eventID", sv.eventID))
	}
	sv.state = StateVarConsensusStateConsensusNotReached
	sv.sendEvent()

}

// consensus was reached either through a vote or through perfect matching of all of 2/3 of the validators.
func (sv *StateVariable) consensusReached(acceptedValue *statevar.KeyValueResult) {
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("consensus reached", logging.String("stateVar", sv.ID), logging.String("eventID", sv.eventID))
	}
	acceptedValue.Validity = statevar.StateValidityConsensus
	sv.state = StateVarConsensusStateConsensusReached
	sv.result(acceptedValue)
	sv.sendEvent()

	// reset the state
	sv.eventID = ""
	sv.validatorResults = nil
	sv.currentValue = acceptedValue
}

func (sv *StateVariable) sendEvent() {
	sv.broker.Send(events.NewStateVarEvent(context.Background(), sv.ID, sv.eventID, stateToName[sv.state]))
}
