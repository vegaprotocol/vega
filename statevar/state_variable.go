package statevar

import (
	"context"
	"math/rand"
	"sort"
	"time"

	vegapb "code.vegaprotocol.io/protos/vega"
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
	StateVarConsensusStateUnspecified StateVarConsensusState = iota
	StateVarConsensusStateCalculationStarted
	StateVarConsensusStatePerfectMatch
	StateVarConsensusStateSeekingConsensus
	StateVarConsensusStateConsensusReached
	StateVarConsensusStateCalculationAborted
	StateVarConsensusStateError
)

var stateToName = map[StateVarConsensusState]string{
	StateVarConsensusStateUnspecified:        "undefined",
	StateVarConsensusStateCalculationStarted: "consensus_calc_started",
	StateVarConsensusStatePerfectMatch:       "perfect_match",
	StateVarConsensusStateSeekingConsensus:   "seeking_consensus",
	StateVarConsensusStateConsensusReached:   "consensus_reached",
	StateVarConsensusStateCalculationAborted: "consensus_calc_aborted",
	StateVarConsensusStateError:              "error",
}

type StateVariable struct {
	log              *logging.Logger
	top              Topology
	cmd              Commander
	broker           Broker
	ID               string                                                    // the unique identifier of the state variable
	converter        statevar.Converter                                        // convert to/from the key/value bundle model into typed result model
	startCalculation func(string, statevar.FinaliseCalculation)                // a callback to the owner to start the calculation of the value of the state variable
	trigger          []statevar.StateVarEventType                              // events that should trigger the calculation of the state variable
	frequency        time.Duration                                             // the frequency for time based triggering
	result           func(context.Context, statevar.StateVariableResult) error // a callback to be called when the value reaches consensus

	nextTimeToRun    time.Time                           // the next scheduled calculation
	state            StateVarConsensusState              // the current status of consensus
	eventID          string                              // the event ID triggering the calculation
	validatorResults map[string]*statevar.KeyValueBundle // the result of the calculation as received from validators
}

func NewStateVar(log *logging.Logger, broker Broker, top Topology, cmd Commander, currentTime time.Time, ID string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.StateVarEventType, frequency time.Duration, result func(context.Context, statevar.StateVariableResult) error) *StateVariable {
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
		converter:        converter,
		startCalculation: startCalculation,
		trigger:          trigger,
		result:           result,
		state:            StateVarConsensusStateUnspecified,
		validatorResults: map[string]*statevar.KeyValueBundle{},
		nextTimeToRun:    nextTimeToRun,
		frequency:        frequency,
	}
	return sv
}

// calculation is required for the state variable for the given event id.
func (sv *StateVariable) eventTriggered(eventID string) error {
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("event triggered", logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}

	// if we get a new event while processing an existing event we abort the current calculation and start a new one
	if sv.eventID != "" {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("aborting state variable event", logging.String("state-var", sv.ID), logging.String("aborted-event-id", sv.eventID), logging.String("new-event-id", sv.eventID))
		}
		sv.state = StateVarConsensusStateCalculationAborted
		sv.sendEvent()
	}

	// reset any existing state
	sv.eventID = eventID
	sv.validatorResults = map[string]*statevar.KeyValueBundle{}
	sv.state = StateVarConsensusStateCalculationStarted
	sv.sendEvent()

	if !sv.top.IsValidator() {
		return nil
	}

	// kickoff calculation
	sv.startCalculation(sv.eventID, sv)

	return nil
}

// CalculationFinished is called from the owner when the calculation is completed to kick off consensus.
func (sv *StateVariable) CalculationFinished(eventID string, result statevar.StateVariableResult, err error) {
	if sv.eventID != eventID {
		sv.log.Warn("ignoring recevied the result of a calculation of an old eventID", logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}

	if err != nil {
		sv.log.Error("could not calculate state for", logging.String("id", sv.ID), logging.String("event-id", eventID))
		sv.state = StateVarConsensusStateError
		sv.sendEvent()
		sv.eventID = ""
		return
	}

	// save our result and send the result to vega to be updated by other nodes.
	svp := &commandspb.StateVariableProposal{
		Proposal: &vegapb.StateValueProposal{
			StateVarId: sv.ID,
			EventId:    sv.eventID,
			Kvb:        sv.converter.InterfaceToBundle(result).ToProto(),
		},
	}
	sv.cmd.Command(context.Background(), txn.StateVariableProposalCommand, svp, func(error) {})
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("result calculated and sent to vega", logging.String("validator", sv.top.SelfNodeID()), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}
}

// bundleReceived is called when we get a result from another validator corresponding to a given event ID.
func (sv *StateVariable) bundleReceived(ctx context.Context, nodeID, eventID string, bundle *statevar.KeyValueBundle, rng *rand.Rand, validatorVotesRequired num.Decimal) {
	// if the bundle is received for a stale or wrong event, ignore it
	if sv.eventID != eventID {
		sv.log.Debug("received a result for a stale event", logging.String("ID", sv.ID), logging.String("from-node", nodeID), logging.String("current-even-id", sv.eventID), logging.String("receivedEventID", eventID))
		return
	}

	// if for some reason we received a result from a non validator node, ignore it
	if !sv.top.IsValidatorNodeID(nodeID) {
		sv.log.Debug("state var bundle received from a non validator node - ignoring", logging.String("from-validator", nodeID), logging.String("state-var", sv.ID), logging.String("eventID", eventID))
		return
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var bundle received", logging.String("from-validator", nodeID), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
	}

	if sv.state == StateVarConsensusStatePerfectMatch || sv.state == StateVarConsensusStateConsensusReached {
		sv.log.Debug("state var bundle received, consensus already reached", logging.String("from-validator", nodeID), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
		return
	}

	// save the result from the validator and check if we have a quorum
	sv.validatorResults[nodeID] = bundle
	numResults := int64(len(sv.validatorResults))
	validatorsNum := num.DecimalFromInt64(int64(len(sv.top.AllNodeIDs())))
	requiredNumberOfResults := validatorsNum.Mul(validatorVotesRequired).IntPart()

	if numResults < requiredNumberOfResults {
		if sv.log.GetLevel() <= logging.DebugLevel {
			sv.log.Debug("waiting for more results for state variable consensus check", logging.String("state-var", sv.ID), logging.String("event-id", eventID), logging.Int64("received", numResults), logging.String("out-of", validatorsNum.String()))
		}
		return
	}

	// if we're already in seeking consensus state, no point in checking if all match - suffice checking if there's a majority with matching within tolerance
	if sv.state == StateVarConsensusStateSeekingConsensus {
		sv.tryConsensus(ctx, rng, requiredNumberOfResults)
		return
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var checking consensus (2/3 of the results received", logging.String("from-validator", nodeID), logging.String("state-var", sv.ID), logging.String("event-id", eventID))
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
			sv.state = StateVarConsensusStateSeekingConsensus
			sv.tryConsensus(ctx, rng, validatorsNum.Mul(validatorVotesRequired).IntPart())
			return
		}
	}

	// we are done - happy days!
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var consensus reached through perfect match", logging.String("state-var", sv.ID), logging.String("event-id", eventID), logging.Int("num-results", len(sv.validatorResults)))
	}
	sv.state = StateVarConsensusStatePerfectMatch
	// convert the result to decimal and let the owner of the state variable know
	sv.consensusReached(ctx, result)
}

// if the bundles are not all equal to each other, choose one at random and verify that all others are within tolerance.
func (sv *StateVariable) tryConsensus(ctx context.Context, rng *rand.Rand, requiredMatches int64) {
	// sort the node IDs for determinism
	nodeIDs := make([]string, 0, len(sv.validatorResults))
	for nodeID := range sv.validatorResults {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	alreadyCheckedForTolerance := map[string]struct{}{}

	for {
		if len(alreadyCheckedForTolerance) == len(nodeIDs) {
			break
		}
		nodeID := nodeIDs[rng.Intn(len(nodeIDs))]
		if _, ok := alreadyCheckedForTolerance[nodeID]; ok {
			continue
		}
		alreadyCheckedForTolerance[nodeID] = struct{}{}
		candidateResult := sv.validatorResults[nodeID]
		countMatch := int64(0)
		for _, nID := range nodeIDs {
			if sv.validatorResults[nID].WithinTolerance(candidateResult) {
				countMatch++
			}
		}
		if countMatch >= requiredMatches {
			sv.state = StateVarConsensusStateConsensusReached
			sv.consensusReached(ctx, candidateResult)
			return
		}
	}

	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("state var consensus NOT reached through random selection", logging.String("state-var", sv.ID), logging.String("event-id", sv.eventID), logging.Int("num-results", len(sv.validatorResults)))
	}
}

// consensus was reached either through a vote or through perfect matching of all of 2/3 of the validators.
func (sv *StateVariable) consensusReached(ctx context.Context, acceptedValue *statevar.KeyValueBundle) {
	if sv.log.GetLevel() <= logging.DebugLevel {
		sv.log.Debug("consensus reached", logging.String("state-var", sv.ID), logging.String("event-id", sv.eventID))
	}

	sv.result(ctx, sv.converter.BundleToInterface(acceptedValue))
	sv.sendEvent()

	// reset the state
	sv.eventID = ""
	sv.validatorResults = nil
}

func (sv *StateVariable) sendEvent() {
	sv.broker.Send(events.NewStateVarEvent(context.Background(), sv.ID, sv.eventID, stateToName[sv.state]))
}
