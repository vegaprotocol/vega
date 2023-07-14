// Copyright (c) 2023 Gobalsky Labs Limited
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

package spec

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
)

// OnMatchedData describes the callback function used when an data dource data matches the spec.
type OnMatchedData func(ctx context.Context, data common.Data) error

// SpecPredicate describes the predicate used to filter the subscribers.
// When returning true, all the subscribers associated to the matching
// Spec are collected.
// The order between specs and subscribers is preserved.
type SpecPredicate func(spec Spec) (bool, error)

// SubscriptionPredicate describes the predicate used to check if any
// of the currently existing subscriptions expects the public keys inside
// the incoming Spec object.
type SubscriptionPredicate func(spec Spec) bool

// SubscriptionID is a unique identifier referencing the subscription of an
// OnMatchedData to a Spec.
type SubscriptionID uint64

// Unsubscriber is a closure that is created at subscription step in order to
// provide the ability to unsubscribe at any conveninent moment.
type Unsubscriber func(context.Context, SubscriptionID)

// updatedSubscription wraps all useful information about an updated
// subscription.
type updatedSubscription struct {
	subscriptionID  SubscriptionID
	spec            datasource.Spec
	specActivatedAt time.Time
}

// filterResult describes the result of the filter operation.
type filterResult struct {
	// specIDs lists all the Spec ID that matched the filter
	// predicate.
	specIDs []SpecID
	// subscribers list all the subscribers associated to the matched Spec.
	subscribers []OnMatchedData
}

// hasMatched returns true if filter has matched the predicated.
func (r filterResult) hasMatched() bool {
	return len(r.specIDs) > 0
}

// specSubscriptions wraps the subscribers (in form of OnMatchedData) to
// the Spec.
type specSubscriptions struct {
	mu sync.RWMutex

	lastSubscriptionID SubscriptionID
	subscriptions      []*specSubscription
	// subscriptionsMatrix maps a SubscriptionID to a SpecID to speed up
	// the retrieval of the OnMatchedData into the subscriptions.
	subscriptionsMatrix map[SubscriptionID]SpecID
}

// newSpecSubscriptions initialises the subscription handler.
func newSpecSubscriptions() *specSubscriptions {
	return &specSubscriptions{
		subscriptions:       []*specSubscription{},
		subscriptionsMatrix: map[SubscriptionID]SpecID{},
	}
}

// hasAnySubscribers checks if any of the subscriptions contains public keys that
// match the given ones by the predicate.
// Returns fast on the first match.
func (s *specSubscriptions) hasAnySubscribers(predicate SubscriptionPredicate) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, subscription := range s.subscriptions {
		if predicate(subscription.spec) {
			return true
		}
	}

	return false
}

// filterSubscribers collects the subscribers that match the predicate on the Spec.
// The order between specs and subscribers is preserved.
func (s *specSubscriptions) filterSubscribers(predicate SpecPredicate) (*filterResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &filterResult{
		specIDs:     []SpecID{},
		subscribers: []OnMatchedData{},
	}

	for _, subscription := range s.subscriptions {
		matched, err := predicate(subscription.spec)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		result.specIDs = append(result.specIDs, subscription.spec.id)
		for _, subscriber := range subscription.subscribers {
			result.subscribers = append(result.subscribers, subscriber.cb)
		}
	}
	return result, nil
}

// getSubscription returns the subscription associated to the given SpecID.  Returns the updates subscription and
// true if this is the first subscription to the spec.
func (s *specSubscriptions) addSubscriber(spec Spec, cb OnMatchedData, tm time.Time) (updatedSubscription, bool) {
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
func (s *specSubscriptions) createSubscription(spec Spec, tm time.Time) *specSubscription {
	subscription := newSpecSubscription(spec, tm)
	s.subscriptions = append(s.subscriptions, subscription)
	return subscription
}

// Internal usage.
func (s *specSubscriptions) getSubscription(id SpecID) (int, *specSubscription) {
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

// specSubscription groups all OnMatchedData callbacks by Spec.
type specSubscription struct {
	spec            Spec
	specActivatedAt time.Time
	subscribers     []*specSubscriber
}

type specSubscriber struct {
	id SubscriptionID
	cb OnMatchedData
}

// Internal usage.
func newSpecSubscription(spec Spec, activationTime time.Time) *specSubscription {
	return &specSubscription{
		spec:            spec,
		specActivatedAt: activationTime,
		subscribers:     []*specSubscriber{},
	}
}

// Internal usage.
func (s *specSubscription) addSubscriber(id SubscriptionID, cb OnMatchedData) {
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
// Spec, false otherwise.
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
