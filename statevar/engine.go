package statevar

import (
	"context"
	"errors"
	"math/rand"
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
	// ErrDuplicateStateVar is returned when trying to add a state variable that already exists.
	ErrDuplicateStateVar = errors.New("Duplicate state variable")
)

//mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/statevar Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error))
}

// Broker send events.
type Broker interface {
	Send(event events.Event)
}

// Topology the topology service.
//mockgen -destination mocks/topology_mock.go -package mocks code.vegaprotocol.io/vega/statevar Tolopology
type Topology interface {
	IsValidatorNodeID(nodeID string) bool
	AllNodeIDs() []string
	Get(key string) *validators.ValidatorData
	IsValidator() bool
	SelfNodeID() string
}

// EpochEngine for being notified on epochs
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

// TimeService for being notified on new blocks for time based calculations
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
}

// StateVarEventType enumeration for supported events triggering calculation.
type StateVarEventType int

const (
	// sample events there may be many more

	StateVarEventTypeAuctionUnknown   StateVarEventType = iota
	StateVarEventTypeAuctionEnded                       = iota
	StateVarEventTypeRiskModelChanged                   = iota
)

// Engine is an engine for creating consensus for floaing point "state variables"
type Engine struct {
	log                    *logging.Logger
	config                 Config
	broker                 Broker
	top                    Topology
	rng                    *rand.Rand
	cmd                    Commander
	eventTypeToStateVar    map[StateVarEventType][]*StateVariable
	stateVars              map[string]*StateVariable
	currentTime            time.Time
	validatorVotesRequired num.Decimal
}

// New instantiates the state variable engine.
func New(log *logging.Logger, config Config, broker Broker, top Topology, cmd Commander, epochEngine EpochEngine, ts TimeService) *Engine {
	e := &Engine{
		log:                 log,
		config:              config,
		broker:              broker,
		top:                 top,
		cmd:                 cmd,
		eventTypeToStateVar: map[StateVarEventType][]*StateVariable{},
		stateVars:           map[string]*StateVariable{},
	}
	epochEngine.NotifyOnEpoch(e.onEpochEvent)
	ts.NotifyOnTick(e.OnTimeTick)

	return e
}

func (e *Engine) OnDefaultValidatorsVoteRequiredUpdate(ctx context.Context, f float64) error {
	e.validatorVotesRequired = num.DecimalFromFloat(f)
	return nil
}

// NewEvent triggers calculation of state variables that depend on the event type.
func (e *Engine) NewEvent(eventType StateVarEventType, eventID string) {
	if _, ok := e.eventTypeToStateVar[eventType]; !ok {
		return
	}
	for _, sv := range e.eventTypeToStateVar[eventType] {
		sv.eventTriggered(eventID)
	}
}

// OnTimeTick triggers the calculation of state variables whose next scheduled calculation is due
func (e *Engine) OnTimeTick(ctx context.Context, t time.Time) {
	e.currentTime = t
	for _, sv := range e.stateVars {
		if (sv.nextTimeToRun != time.Time{}) && sv.nextTimeToRun.UnixNano() <= t.UnixNano() {
			sv.eventTriggered(t.Format("20060102_150405.999999999"))
			sv.nextTimeToRun = t.Add(sv.frequency)
		}
	}
}

// OnEpochEvent resets the seed of the rng when a new epoch begins
func (e *Engine) onEpochEvent(ctx context.Context, epoch types.Epoch) {
	if (epoch.EndTime == time.Time{}) {
		e.rng = rand.New(rand.NewSource(epoch.StartTime.Unix()))
	}
}

// AddStateVariable register a new state variable for which consensus should be managed.
// ID - the unique identifier of the state variable
// calculateFunc - a callback for calculating the value of the state variable
// trigger - a slice of events that should trigger the calculation of the state variable
// frequency - if time based triggering the frequency to trigger, Duration(0) for no time based trigger
// result - a callback for storing the result
// defaultValue - the default value to use (as decimal)
func (e *Engine) AddStateVariable(ID string, calculateFunc func() (*statevar.KeyValueBundle, error), trigger []StateVarEventType, frequency time.Duration, result func(*statevar.KeyValueResult) error, defaultValue *statevar.KeyValueResult) error {
	if _, ok := e.stateVars[ID]; ok {
		return ErrDuplicateStateVar
	}

	sv := NewStateVar(e.log, e.broker, e.top, e.cmd, e.currentTime, ID, calculateFunc, trigger, frequency, result, defaultValue)
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
