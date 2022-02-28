package statevar

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
	"code.vegaprotocol.io/vega/validators"

	"github.com/golang/protobuf/proto"
)

var (
	// ErrUnknownStateVar is returned when we get a request (vote, result) for a state variable we don't have.
	ErrUnknownStateVar  = errors.New("unknown state variable")
	ErrNameAlreadyExist = errors.New("state variable already exists with the same name")
	chars               = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// go:generate go run github.com/golang/mock/mockgen -destination -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/statevar Commander.
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error))
}

// Broker send events.
type Broker interface {
	SendBatch(events []events.Event)
}

// Topology the topology service.
// go:generate go run github.com/golang/mock/mockgen -destination -destination mocks/topology_mock.go -package mocks code.vegaprotocol.io/vega/statevar Tolopology.
type Topology interface {
	IsValidatorVegaPubKey(node string) bool
	AllNodeIDs() []string
	Get(key string) *validators.ValidatorData
	IsValidator() bool
	SelfNodeID() string
}

// EpochEngine for being notified on epochs.
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

// TimeService for being notified on new blocks for time based calculations.
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
}

// Engine is an engine for creating consensus for floaing point "state variables".
type Engine struct {
	log                    *logging.Logger
	config                 Config
	broker                 Broker
	top                    Topology
	rng                    *rand.Rand
	cmd                    Commander
	eventTypeToStateVar    map[statevar.StateVarEventType][]*StateVariable
	stateVars              map[string]*StateVariable
	currentTime            time.Time
	validatorVotesRequired num.Decimal
	seq                    int
	updateFrequency        time.Duration
	readyForTimeTrigger    map[string]struct{}
	stateVarToNextCalc     map[string]time.Time
	ss                     *snapshotState
}

// New instantiates the state variable engine.
func New(log *logging.Logger, config Config, broker Broker, top Topology, cmd Commander, ts TimeService) *Engine {
	lg := log.Named(namedLogger)
	lg.SetLevel(config.Level.Get())
	e := &Engine{
		log:                 lg,
		config:              config,
		broker:              broker,
		top:                 top,
		cmd:                 cmd,
		eventTypeToStateVar: map[statevar.StateVarEventType][]*StateVariable{},
		stateVars:           map[string]*StateVariable{},
		seq:                 0,
		readyForTimeTrigger: map[string]struct{}{},
		stateVarToNextCalc:  map[string]time.Time{},
		ss: &snapshotState{
			changed: true,
		},
	}
	ts.NotifyOnTick(e.OnTimeTick)

	return e
}

// generate an id for the variable.
func (e *Engine) generateID(asset, market, name string) string {
	return asset + "_" + market + "_" + name
}

// generate a random event identifier.
func (e *Engine) generateEventID(asset, market string) string {
	// using the pseudorandomness here to avoid saving a sequence number to the snapshot
	b := make([]rune, 32)
	for i := range b {
		b[i] = chars[e.rng.Intn(len(chars))]
	}
	prefix := asset + "_" + market
	e.seq++
	suffix := string(b)
	return prefix + "_" + suffix
}

// OnFloatingPointUpdatesDurationUpdate updates the update frequency from the network parameter.
func (e *Engine) OnFloatingPointUpdatesDurationUpdate(ctx context.Context, updateFrequency time.Duration) error {
	e.log.Info("updating floating point update frequency", logging.String("updateFrequency", updateFrequency.String()))
	e.updateFrequency = updateFrequency
	return nil
}

// OnDefaultValidatorsVoteRequiredUpdate updates the required majority for a vote on a proposed value.
func (e *Engine) OnDefaultValidatorsVoteRequiredUpdate(ctx context.Context, d num.Decimal) error {
	e.validatorVotesRequired = d
	e.log.Info("ValidatorsVoteRequired updated", logging.String("validatorVotesRequired", e.validatorVotesRequired.String()))
	return nil
}

// NewEvent triggers calculation of state variables that depend on the event type.
func (e *Engine) NewEvent(asset, market string, eventType statevar.StateVarEventType) {
	// disabling for now until wiring all state variables
	// if _, ok := e.eventTypeToStateVar[eventType]; !ok {
	// 	e.log.Panic("Unexpected event received", logging.Int("event-type", int(eventType)), logging.String("asset", asset), logging.String("market", market))
	// }
	// generate a unique event id
	eventID := e.generateEventID(asset, market)
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("New event for state variable received", logging.String("eventID", eventID), logging.String("asset", asset), logging.String("market", market))
	}

	for _, sv := range e.eventTypeToStateVar[eventType] {
		if sv.market != market || sv.asset != asset {
			continue
		}
		sv.eventTriggered(eventID)
		// if the sv is time triggered - reset the next run to be now + frequency
		if _, ok := e.stateVarToNextCalc[sv.ID]; ok {
			e.stateVarToNextCalc[sv.ID] = e.currentTime.Add(e.updateFrequency)
			e.ss.changed = true
		}
	}
}

// OnTimeTick triggers the calculation of state variables whose next scheduled calculation is due.
func (e *Engine) OnTimeTick(ctx context.Context, t time.Time) {
	e.currentTime = t
	e.rng = rand.New(rand.NewSource(t.Unix()))

	// update all state vars on the new block so they can send the batch of events from the previous block
	allStateVarIDs := make([]string, 0, len(e.stateVars))
	for ID := range e.stateVars {
		allStateVarIDs = append(allStateVarIDs, ID)
	}
	sort.Strings(allStateVarIDs)

	for _, ID := range allStateVarIDs {
		e.stateVars[ID].startBlock(ctx)
	}

	// get all the state var with time triggers whose time to tick has come and call them
	stateVarIDs := []string{}
	for ID, nextTime := range e.stateVarToNextCalc {
		if nextTime.UnixNano() <= t.UnixNano() {
			stateVarIDs = append(stateVarIDs, ID)
		}
	}

	sort.Strings(stateVarIDs)
	eventID := t.Format("20060102_150405.999999999")
	for _, ID := range stateVarIDs {
		sv := e.stateVars[ID]

		if e.log.GetLevel() <= logging.DebugLevel {
			e.log.Debug("New time based event for state variable received", logging.String("eventID", eventID))
		}
		sv.eventTriggered(eventID)
		e.stateVarToNextCalc[ID] = t.Add(e.updateFrequency)
		e.ss.changed = true
	}
}

// ReadyForTimeTrigger is called when the market is ready for time triggered event and sets the next time to run for all state variables of that market that are time triggered.
// This is expected to be called at the end of the opening auction for the market.
func (e *Engine) ReadyForTimeTrigger(asset, mktID string) {
	e.log.Info("ReadyForTimeTrigger", logging.String("asset", asset), logging.String("market-id", mktID))
	if _, ok := e.readyForTimeTrigger[asset+mktID]; !ok {
		e.readyForTimeTrigger[asset+mktID] = struct{}{}
		for _, sv := range e.eventTypeToStateVar[statevar.StateVarEventTypeTimeTrigger] {
			if sv.asset == asset && sv.market == mktID {
				e.stateVarToNextCalc[sv.ID] = e.currentTime.Add(e.updateFrequency)
				e.ss.changed = true
			}
		}
	}
}

// RegisterStateVariable register a new state variable for which consensus should be managed.
// converter - converts from the native format of the variable and the key value bundle format and vice versa
// startCalculation - a callback to trigger an asynchronous state var calc - the result of which is given through the FinaliseCalculation interface
// trigger - a slice of events that should trigger the calculation of the state variable
// frequency - if time based triggering the frequency to trigger, Duration(0) for no time based trigger
// result - a callback for returning the result converted to the native structure.
func (e *Engine) RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.StateVarEventType, result func(context.Context, statevar.StateVariableResult) error) error {
	ID := e.generateID(asset, market, name)
	if _, ok := e.stateVars[ID]; ok {
		return ErrNameAlreadyExist
	}

	e.log.Info("added state variable", logging.String("id", ID), logging.String("asset", asset), logging.String("market", market))

	sv := NewStateVar(e.log, e.broker, e.top, e.cmd, e.currentTime, ID, asset, market, converter, startCalculation, trigger, result)
	e.stateVars[ID] = sv
	for _, t := range trigger {
		if _, ok := e.eventTypeToStateVar[t]; !ok {
			e.eventTypeToStateVar[t] = []*StateVariable{}
		}
		e.eventTypeToStateVar[t] = append(e.eventTypeToStateVar[t], sv)
	}
	return nil
}

// ProposedValueReceived is called when we receive a result from another node with a proposed result for the calculation triggered by an event.
func (e *Engine) ProposedValueReceived(ctx context.Context, ID, nodeID, eventID string, bundle *statevar.KeyValueBundle) error {
	e.log.Info("bundle received", logging.String("id", ID), logging.String("from-node", nodeID), logging.String("event-id", eventID))

	if sv, ok := e.stateVars[ID]; ok {
		sv.bundleReceived(ctx, e.currentTime, nodeID, eventID, bundle, e.rng, e.validatorVotesRequired)
		return nil
	}
	e.log.Error("ProposedValueReceived called with unknown var", logging.String("id", ID), logging.String("from-node", nodeID))
	return ErrUnknownStateVar
}
