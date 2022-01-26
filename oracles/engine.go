package oracles

import (
	"context"
	"fmt"
	"time"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

const (
	InternalOraclePrefix    = "vegaprotocol.internal"
	InternalOracleTimestamp = InternalOraclePrefix + ".timestamp"
)

// Broker no need to mock (use broker package mock).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/oracles TimeService
type TimeService interface {
	NotifyOnTick(f func(context.Context, time.Time))
	NotifyInternalOracleTimestamp(f func(context.Context, OracleData))
}

// Engine is responsible of broadcasting the OracleData to products and risk
// models interested in it.
type Engine struct {
	log           *logging.Logger
	broker        Broker
	CurrentTime   time.Time
	subscriptions specSubscriptions
}

// NewEngine creates a new oracle Engine.
func NewEngine(
	log *logging.Logger,
	conf Config,
	currentTime time.Time,
	broker Broker,
	ts TimeService,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	e := &Engine{
		log:           log,
		broker:        broker,
		CurrentTime:   currentTime,
		subscriptions: newSpecSubscriptions(),
	}

	ts.NotifyOnTick(e.UpdateCurrentTime)
	ts.NotifyInternalOracleTimestamp(e.BroadcastVegaTime)
	return e
}

// UpdateCurrentTime listens to update of the current Vega time.
func (e *Engine) UpdateCurrentTime(ctx context.Context, ts time.Time) {
	e.CurrentTime = ts
}

func (e *Engine) BroadcastVegaTime(ctx context.Context, data OracleData) {
	err := e.sendOracleUpdate(ctx, data)
	if err != nil {
		e.log.Debug("broadcasting internal vega timestamp", logging.Error(err))
	}
}

func (e *Engine) sendOracleUpdate(ctx context.Context, data OracleData) error {
	result, err := e.subscriptions.filterSubscribers(func(spec OracleSpec) (bool, error) {
		return spec.MatchData(data)
	})
	if err != nil {
		e.log.Debug("error in filtering subscribers",
			logging.Error(err),
		)
		return err
	}

	if result.hasMatched() {
		for _, subscriber := range result.subscribers {
			if err := subscriber(ctx, data); err != nil {
				e.log.Debug("broadcasting data to subscriber failed",
					logging.Error(err),
				)
			}
		}
		e.sendOracleDataBroadcast(ctx, data, result.oracleSpecIDs)
	}

	return nil
}

// BroadcastData broadcasts data to products and risk models that are interested in it. If no one is listening to this OracleData, it is discarded.
func (e *Engine) BroadcastData(ctx context.Context, data OracleData) error {
	err := e.sendOracleUpdate(ctx, data)
	if err != nil {
		e.log.Debug("failed to send oracle update",
			logging.Error(err),
		)
	}
	return err
}

// Subscribe registers a callback for a given OracleSpec that is call when an
// OracleData matches the spec.
// It returns a SubscriptionID that is used to Unsubscribe.
// If cb is nil, the method panics.
func (e *Engine) Subscribe(ctx context.Context, spec OracleSpec, cb OnMatchedOracleData) SubscriptionID {
	if cb == nil {
		panic(fmt.Sprintf("a callback is required for spec %v", spec))
	}
	updatedSubscription := e.subscriptions.addSubscriber(spec, cb, e.CurrentTime)
	e.sendNewOracleSpecSubscription(ctx, updatedSubscription)
	return updatedSubscription.subscriptionID
}

// Unsubscribe unregisters the callback associated to the SubscriptionID.
// If the id doesn't exist, this method panics.
func (e *Engine) Unsubscribe(ctx context.Context, id SubscriptionID) {
	updatedSubscription, hasNoMoreSubscriber := e.subscriptions.removeSubscriber(id)
	if hasNoMoreSubscriber {
		e.sendOracleSpecDeactivation(ctx, updatedSubscription)
	}
}

// sendNewOracleSpecSubscription send an event to the broker to inform of the
// subscription (and thus activation) to an oracle spec.
// This may be a subscription to a brand new oracle spec, or an additional one.
func (e *Engine) sendNewOracleSpecSubscription(ctx context.Context, update updatedSubscription) {
	specAsProto := update.specProto
	specAsProto.CreatedAt = update.specActivatedAt.UnixNano()
	specAsProto.Status = oraclespb.OracleSpec_STATUS_ACTIVE
	e.broker.Send(events.NewOracleSpecEvent(ctx, specAsProto))
}

// sendOracleSpecDeactivation send an event to the broker to inform of
// the deactivation (and thus activation) to an oracle spec.
// This may be a subscription to a brand new oracle spec, or an additional one.
func (e *Engine) sendOracleSpecDeactivation(ctx context.Context, update updatedSubscription) {
	specAsProto := update.specProto
	specAsProto.CreatedAt = update.specActivatedAt.UnixNano()
	specAsProto.Status = oraclespb.OracleSpec_STATUS_DEACTIVATED
	e.broker.Send(events.NewOracleSpecEvent(ctx, specAsProto))
}

// sendOracleSpecDeactivation send an event to the broker to inform of
// the deactivation (and thus activation) to an oracle spec.
// This may be a subscription to a brand new oracle spec, or an additional one.
func (e *Engine) sendOracleDataBroadcast(ctx context.Context, data OracleData, specIDs []OracleSpecID) {
	payload := []*oraclespb.Property{}
	for name, value := range data.Data {
		payload = append(payload, &oraclespb.Property{
			Name:  name,
			Value: value,
		})
	}

	ids := []string{}
	for _, specID := range specIDs {
		ids = append(ids, string(specID))
	}

	dataProto := oraclespb.OracleData{
		PubKeys:        data.PubKeys,
		Data:           payload,
		MatchedSpecIds: ids,
		BroadcastAt:    e.CurrentTime.UnixNano(),
	}
	e.broker.Send(events.NewOracleDataEvent(ctx, dataProto))
}
