package oracles

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

// OnMatchedOracleData describes the callback function used to received
type OnMatchedOracleData func(ctx context.Context, data OracleData) error

// OracleSpecPredicate describes the predicate used to filter the subscribers.
// When returning true, all the subscribers associated to the matching
// OracleSpec are collected.
// The order between specs and subscribers is preserved.
type OracleSpecPredicate func(spec OracleSpec) (bool, error)

// SubscriptionID is a unique identifier referencing the subscription of an
// OnMatchedOracleData to an OracleSpec.
type SubscriptionID uint64

// updatedSubscription wraps all useful information about an updated
// subscription.
type updatedSubscription struct {
	subscriptionID  SubscriptionID
	specProto       oraclespb.OracleSpec
	specActivatedAt time.Time
}

// filterResult describes the result of the filter operation.
type filterResult struct {
	// oracleSpecIDs lists all the OracleSpec ID that matched the filter
	// predicate.
	oracleSpecIDs []OracleSpecID
	// subscribers list all the subscribers associated to the matched
	// OracleSpec.
	subscribers []OnMatchedOracleData
}

// hasMatched returns true if filter has matched the predicated.
func (r filterResult) hasMatched() bool {
	return len(r.oracleSpecIDs) > 0
}

// specSubscriptions wraps the subscribers (in form of OnMatchedOracleData) to
// the OracleSpec.
type specSubscriptions struct {
	lastSubscriptionID SubscriptionID
	subscriptions      []*specSubscription
	// subscriptionsMatrix maps a SubscriptionID to an OracleSpecID to speed up
	// the retrieval of the OnMatchedOracleData into the subscriptions.
	subscriptionsMatrix map[SubscriptionID]OracleSpecID
}

// newSpecSubscriptions initialises the subscription handler.
func newSpecSubscriptions() specSubscriptions {
	return specSubscriptions{
		subscriptions:       []*specSubscription{},
		subscriptionsMatrix: map[SubscriptionID]OracleSpecID{},
	}
}

// filterSubscribers collects the subscribers that match the predicate on the
// OracleSpec.
// The order between specs and subscribers is preserved.
func (s specSubscriptions) filterSubscribers(predicate OracleSpecPredicate) (*filterResult, error) {
	result := &filterResult{
		oracleSpecIDs: []OracleSpecID{},
		subscribers:   []OnMatchedOracleData{},
	}

	for _, subscription := range s.subscriptions {
		matched, err := predicate(subscription.spec)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		result.oracleSpecIDs = append(result.oracleSpecIDs, subscription.spec.id)
		for _, subscriber := range subscription.subscribers {
			result.subscribers = append(result.subscribers, subscriber.cb)
		}
	}
	return result, nil
}

func (s *specSubscriptions) addSubscriber(spec OracleSpec, cb OnMatchedOracleData, currentTime time.Time) updatedSubscription {
	_, subscription := s.getSubscription(spec.id)
	if subscription == nil {
		subscription = s.createSubscription(spec, currentTime)
	}

	subscriptionID := s.nextSubscriptionID()
	subscription.addSubscriber(subscriptionID, cb)

	s.subscriptionsMatrix[subscriptionID] = spec.id

	return updatedSubscription{
		subscriptionID:  subscriptionID,
		specActivatedAt: subscription.specActivatedAt,
		specProto:       spec.Proto,
	}
}

func (s *specSubscriptions) removeSubscriber(subscriptionID SubscriptionID) (updatedSubscription, bool) {
	specID, ok := s.subscriptionsMatrix[subscriptionID]
	if !ok {
		panic(fmt.Sprintf("unknown subscriber ID %d", subscriptionID))
	}

	index, subscription := s.getSubscription(specID)
	subscription.removeSubscriber(subscriptionID)

	delete(s.subscriptionsMatrix, subscriptionID)

	hasNoMoreSubscriber := subscription.hasNoMoreSubscriber()
	if hasNoMoreSubscriber {
		s.removeSubscriptionFromIndex(index)
	}

	return updatedSubscription{
		subscriptionID:  subscriptionID,
		specActivatedAt: subscription.specActivatedAt,
		specProto:       subscription.spec.Proto,
	}, hasNoMoreSubscriber
}

// Internal usage.
func (s *specSubscriptions) removeSubscriptionFromIndex(index int) {
	copy(s.subscriptions[index:], s.subscriptions[index+1:])
	lastIndex := len(s.subscriptions) - 1
	s.subscriptions[lastIndex] = nil
	s.subscriptions = s.subscriptions[:lastIndex]
}

// Internal usage.
func (s *specSubscriptions) createSubscription(spec OracleSpec, currentTime time.Time) *specSubscription {
	subscription := newOracleSpecSubscription(spec, currentTime)
	s.subscriptions = append(s.subscriptions, subscription)
	return subscription
}

// Internal usage.
func (s *specSubscriptions) getSubscription(id OracleSpecID) (int, *specSubscription) {
	for i, subscription := range s.subscriptions {
		if subscription.spec.id == id {
			return i, subscription
		}
	}
	return -1, nil
}

// nextSubscriptionID computes the next SubscriptionID
// Internal usage.
func (s *specSubscriptions) nextSubscriptionID() SubscriptionID {
	return SubscriptionID(
		atomic.AddUint64((*uint64)(&s.lastSubscriptionID), 1),
	)
}

// specSubscription groups all OnMatchedOracleData callbacks by
// OracleSpec.
type specSubscription struct {
	spec            OracleSpec
	specActivatedAt time.Time
	subscribers     []*specSubscriber
}

type specSubscriber struct {
	id SubscriptionID
	cb OnMatchedOracleData
}

// Internal usage.
func newOracleSpecSubscription(spec OracleSpec, activationTime time.Time) *specSubscription {
	return &specSubscription{
		spec:            spec,
		specActivatedAt: activationTime,
		subscribers:     []*specSubscriber{},
	}
}

// Internal usage.
func (s *specSubscription) addSubscriber(id SubscriptionID, cb OnMatchedOracleData) {
	s.subscribers = append(s.subscribers, &specSubscriber{
		id: id,
		cb: cb,
	})
}

// Internal usage.
func (s *specSubscription) removeSubscriber(id SubscriptionID) {
	index, _ := s.getSubscriber(id)
	s.removeSubscriberFromIndex(index)
}

// hasNoMoreSubscriber returns true if there is no subscriber for the associated
// OracleSpec, false otherwise.
// Internal usage.
func (s *specSubscription) hasNoMoreSubscriber() bool {
	return len(s.subscribers) == 0
}

// Internal usage.
func (s *specSubscription) getSubscriber(id SubscriptionID) (int, *specSubscriber) {
	for i, subscriber := range s.subscribers {
		if subscriber.id == id {
			return i, subscriber
		}
	}
	return -1, nil
}

// Internal usage.
func (s *specSubscription) removeSubscriberFromIndex(index int) {
	copy(s.subscribers[index:], s.subscribers[index+1:])
	lastIndex := len(s.subscribers) - 1
	s.subscribers[lastIndex] = nil
	s.subscribers = s.subscribers[:lastIndex]
}
