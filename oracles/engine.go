package oracles

import (
	"context"
	"fmt"
	"sync/atomic"
)

type OnMatchedOracleData func(ctx context.Context, data OracleData)
type SubscriptionID uint64

// OracleData holds normalized data coming from an oracle.
type OracleData struct {
	PubKeys []string
	Data    map[string]string
}

// Engine is responsible of broadcasting the OracleData to products and risk
// models interested in it.
type Engine struct {
	lastSubscriptionID SubscriptionID
	// TODO Using a map is not deterministic. Should be a list.
	subscribers        map[SubscriptionID]oracleSpecSubscriber
}

// oracleSpecSubscriber groups a OnMatchedOracleData callback to its
// oraclesv1.OracleSpec.
type oracleSpecSubscriber struct {
	oracleSpec OracleSpec
	callback   OnMatchedOracleData
}

// NewEngine creates a new oracle Engine.
func NewEngine() *Engine {
	return &Engine{
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
			subscriber.callback(ctx, data)
		}
	}
	return nil
}

// Subscribe registers a callback for a given oraclesv1.OracleSpec that is call
// when an OracleData matches the spec.
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
