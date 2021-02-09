package oracles

import (
	"context"
	"fmt"
	"sync/atomic"

	"code.vegaprotocol.io/vega/logging"
)

type OnMatchedOracleData func(ctx context.Context, data OracleData) error
type SubscriptionID uint64

// Engine is responsible of broadcasting the OracleData to products and risk
// models interested in it.
type Engine struct {
	log                *logging.Logger
	lastSubscriptionID SubscriptionID
	// TODO Using a map is not deterministic. Should be a list.
	subscribers map[SubscriptionID]oracleSpecSubscriber
}

// oracleSpecSubscriber groups a OnMatchedOracleData callback to its
// oraclesv1.OracleSpecConfiguration.
type oracleSpecSubscriber struct {
	oracleSpec OracleSpec
	callback   OnMatchedOracleData
}

// NewEngine creates a new oracle Engine.
func NewEngine(log *logging.Logger, conf Config) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	return &Engine{
		log:         log,
		subscribers: make(map[SubscriptionID]oracleSpecSubscriber),
	}
}

// BroadcastData broadcasts the OracleData to products and risk models that are
// interested in it. If no one is listening to this OracleData, it is discarded.
func (e *Engine) BroadcastData(ctx context.Context, data OracleData) error {
	for _, subscriber := range e.subscribers {
		matched, err := subscriber.oracleSpec.MatchData(data)
		if err != nil {
			return err
		}
		if matched {
			if err := subscriber.callback(ctx, data); err != nil {
				e.log.Error(
					"broadcasting data to subscriber failed",
					logging.Error(err),
				)
				return err
			}
		}
	}
	return nil
}

// Subscribe registers a callback for a given oraclesv1.OracleSpecConfiguration
// that is call when an OracleData matches the spec.
// It returns a SubscriptionID that is used to Unsubscribe.
// If cb is nil, the method panics.
func (e *Engine) Subscribe(spec OracleSpec, cb OnMatchedOracleData) SubscriptionID {
	if cb == nil {
		panic(fmt.Sprintf("a callback is required for spec %v", spec))
	}

	id := e.nextSubscriptionID()

	e.subscribers[id] = oracleSpecSubscriber{
		oracleSpec: spec,
		callback:   cb,
	}

	return id
}

// Unsubscribe unregisters the callback associated to the SubscriptionID.
// If the id doesn't exist, this method panics.
func (e *Engine) Unsubscribe(id SubscriptionID) {
	if _, ok := e.subscribers[id]; !ok {
		panic(fmt.Sprintf("unknown subscriber ID %d", id))
	}

	delete(e.subscribers, id)
}

// nextSubscriptionID computes the next SubscriptionID
func (e *Engine) nextSubscriptionID() SubscriptionID {
	return SubscriptionID(
		atomic.AddUint64((*uint64)(&e.lastSubscriptionID), 1),
	)
}
