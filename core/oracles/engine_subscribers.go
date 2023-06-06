// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package oracles

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"
)

// OnMatchedOracleData describes the callback function used when an oracle data matches the spec.
type OnMatchedOracleData func(ctx context.Context, data OracleData) error

// OracleSpecPredicate describes the predicate used to filter the subscribers.
// When returning true, all the subscribers associated to the matching
// OracleSpec are collected.
// The order between specs and subscribers is preserved.
type OracleSpecPredicate func(spec OracleSpec) (bool, error)

// OracleSubscriptionPredicate describes the predicate used to check if any
// of the currently existing subscriptions expects the public keys inside
// the incoming OracleSpec object.
type OracleSubscriptionPredicate func(spec OracleSpec) bool

// SubscriptionID is a unique identifier referencing the subscription of an
// OnMatchedOracleData to an OracleSpec.
type SubscriptionID uint64

// Unsubscriber is a closure that is created at subscription step in order to
// provide the ability to unsubscribe at any conveninent moment.
type Unsubscriber func(context.Context, SubscriptionID)

// updatedSubscription wraps all useful information about an updated
// subscription.
type updatedSubscription struct {
	subscriptionID  SubscriptionID
	spec            types.OracleSpec
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
	mu sync.RWMutex

	lastSubscriptionID SubscriptionID
	subscriptions      []*specSubscription
	// subscriptionsMatrix maps a SubscriptionID to an OracleSpecID to speed up
	// the retrieval of the OnMatchedOracleData into the subscriptions.
	subscriptionsMatrix map[SubscriptionID]OracleSpecID
}

// newSpecSubscriptions initialises the subscription handler.
func newSpecSubscriptions() *specSubscriptions {
	return &specSubscriptions{
		subscriptions:       []*specSubscription{},
		subscriptionsMatrix: map[SubscriptionID]OracleSpecID{},
	}
}

// hasAnySubscribers checks if any of the subscriptions contains public keys that
// match the given ones by the predicate.
// Returns fast on the first match.
func (s *specSubscriptions) hasAnySubscribers(predicate OracleSubscriptionPredicate) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, subscription := range s.subscriptions {
		if predicate(subscription.spec) {
			return true
		}
	}

	return false
}

// filterSubscribers collects the subscribers that match the predicate on the
// OracleSpec.
// The order between specs and subscribers is preserved.
func (s *specSubscriptions) filterSubscribers(predicate OracleSpecPredicate) (*filterResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

// getSubscription returns the subscription associated to the given OracleSpecID.  Returns the updates subscription and
// true if this is the first subscription to the spec.
func (s *specSubscriptions) addSubscriber(spec OracleSpec, cb OnMatchedOracleData, tm time.Time) (updatedSubscription, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	firstSubscription := false
	_, subscription := s.getSubscription(spec.id)
	if subscription == nil {
		firstSubscription = true
		subscription = s.createSubscription(spec, tm)
	}

	subscriptionID := s.nextSubscriptionID()
	subscription.addSubscriber(subscriptionID, cb)

	s.subscriptionsMatrix[subscriptionID] = spec.id

	return updatedSubscription{
		subscriptionID:  subscriptionID,
		specActivatedAt: subscription.specActivatedAt,
		spec:            *spec.OriginalSpec,
	}, firstSubscription
}

func (s *specSubscriptions) removeSubscriber(subscriptionID SubscriptionID) (updatedSubscription, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		spec:            *subscription.spec.OriginalSpec,
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
func (s *specSubscriptions) createSubscription(spec OracleSpec, tm time.Time) *specSubscription {
	subscription := newOracleSpecSubscription(spec, tm)
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
	s.lastSubscriptionID++
	return s.lastSubscriptionID
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
