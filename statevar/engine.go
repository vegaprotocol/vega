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
	ErrUnknownStateVar = errors.New("unknown state variable")
	chars              = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// go:generate go run github.com/golang/mock/mockgem -destination -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/statevar Commander.
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error))
}

// Broker send events.
type Broker interface {
	Send(event events.Event)
}

// Topology the topology service.
// go:generate go run github.com/golang/mock/mockgem -destination -destination mocks/topology_mock.go -package mocks code.vegaprotocol.io/vega/statevar Tolopology.
type Topology interface {
	IsValidatorNodeID(nodeID string) bool
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
}

// New instantiates the state variable engine.
func New(log *logging.Logger, config Config, broker Broker, top Topology, cmd Commander, epochEngine EpochEngine, ts TimeService) *Engine {
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
	}
	epochEngine.NotifyOnEpoch(e.OnEpochEvent)
	ts.NotifyOnTick(e.OnTimeTick)

	return e
}

func (e *Engine) variableID() string {
	b := make([]rune, 32)
	for i := range b {
		b[i] = chars[e.rng.Intn(len(chars))]
	}
	return string(b)
}

func (e *Engine) OnDefaultValidatorsVoteRequiredUpdate(ctx context.Context, f float64) error {
	e.validatorVotesRequired = num.DecimalFromFloat(f)
	e.log.Info("ValidatorsVoteRequired updated", logging.String("validatorVotesRequired", e.validatorVotesRequired.String()))
	return nil
}

// NewEvent triggers calculation of state variables that depend on the event type.
func (e *Engine) NewEvent(eventType statevar.StateVarEventType, eventID string) {
	if _, ok := e.eventTypeToStateVar[eventType]; !ok {
		e.log.Panic("Unexpected event received", logging.Int("event-type", int(eventType)), logging.String("event-id", eventID))
	}

	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("New event for state variable received", logging.String("eventID", eventID))
	}

	for _, sv := range e.eventTypeToStateVar[eventType] {
		sv.eventTriggered(eventID)
	}
}

// OnTimeTick triggers the calculation of state variables whose next scheduled calculation is due.
func (e *Engine) OnTimeTick(ctx context.Context, t time.Time) {
	e.currentTime = t

	stateVarIDs := []string{}
	for ID, sv := range e.stateVars {
		if (sv.nextTimeToRun != time.Time{}) && sv.nextTimeToRun.UnixNano() <= t.UnixNano() {
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
		sv.nextTimeToRun = t.Add(sv.frequency)
	}
}

// OnEpochEvent resets the seed of the rng when a new epoch begins.
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if (epoch.EndTime == time.Time{}) {
		e.rng = rand.New(rand.NewSource(epoch.StartTime.Unix()))
	}
}

// AddStateVariable register a new state variable for which consensus should be managed.
// converter - converts from the native format of the variable and the key value bundle format and vice versa
// startCalculation - a callback to trigger an asynchronous state var calc - the result of which is given through the FinaliseCalculation interface
// trigger - a slice of events that should trigger the calculation of the state variable
// frequency - if time based triggering the frequency to trigger, Duration(0) for no time based trigger
// result - a callback for returning the result converted to the desired structure
func (e *Engine) AddStateVariable(converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.StateVarEventType, frequency time.Duration, result func(statevar.StateVariableResult) error) error {
	ID := e.variableID()

	sv := NewStateVar(e.log, e.broker, e.top, e.cmd, e.currentTime, ID, converter, startCalculation, trigger, frequency, result)
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
	if sv, ok := e.stateVars[ID]; ok {
		sv.bundleReceived(nodeID, eventID, bundle, e.rng, e.validatorVotesRequired)
		return nil
	}
	return ErrUnknownStateVar
}
