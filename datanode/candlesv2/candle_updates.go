// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package candlesv2

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

var ErrNewSubscriberNotReady = errors.New("new subscriber was not ready to receive the last candle data")

type candleSource interface {
	GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
		p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error)
}

type subscriptionMsg struct {
	subscribe bool
	id        string
	out       chan entities.Candle
}

func (m subscriptionMsg) String() string {
	if m.subscribe {
		return fmt.Sprintf("unsubscribe, subscription id:%s", m.id)
	}

	return "subscribe"
}

type CandleUpdates struct {
	log                *logging.Logger
	candleSource       candleSource
	candleID           string
	subscriptionMsgCh  chan subscriptionMsg
	nextSubscriptionID atomic.Uint64
	config             CandleUpdatesConfig
	subs               map[string]chan entities.Candle
	mu                 *sync.RWMutex
	lastCandle         *entities.Candle
}

func NewCandleUpdates(ctx context.Context, log *logging.Logger, candleID string, candleSource candleSource,
	config CandleUpdatesConfig,
) *CandleUpdates {
	ces := &CandleUpdates{
		log:               log,
		candleSource:      candleSource,
		candleID:          candleID,
		config:            config,
		subscriptionMsgCh: make(chan subscriptionMsg, config.CandleUpdatesStreamSubscriptionMsgBufferSize),
		subs:              map[string]chan entities.Candle{},
		mu:                &sync.RWMutex{},
	}

	go ces.run(ctx)

	return ces
}

func (s *CandleUpdates) run(ctx context.Context) {
	defer s.closeAllSubscriptions()

	ticker := time.NewTicker(s.config.CandleUpdatesStreamInterval.Duration)
	defer ticker.Stop()

	candleUpdatesFailed := false
	updateCandles := func(now time.Time) *entities.Candle {
		// no subscriptions, don't update candles and remove last candle.
		if len(s.subs) == 0 {
			return nil
		}
		candles, err := s.getCandleUpdates(ctx, now)
		if err != nil {
			if !candleUpdatesFailed {
				s.log.Error("Failed to get candles for candle id", logging.String("candle", s.candleID), logging.Error(err))
			}
			candleUpdatesFailed = true
			return s.lastCandle // keep last candle we successfully obtained
		}
		if candleUpdatesFailed {
			s.log.Info("Successfully got candles for candle id", logging.String("candle", s.candleID))
			candleUpdatesFailed = false
		}
		if len(candles) == 0 {
			return s.lastCandle // no new data, just keep the reference to the last candle we had
		}
		// send the new data to all subscribers.
		_ = s.sendCandlesToSubscribers(candles, s.subs)
		// find the most recent, non zero candle as the last candle we know exists
		for i := len(candles) - 1; i >= 0; i-- {
			last := candles[i]
			if !last.High.IsZero() && !last.Low.IsZero() {
				return &last
			}
		}
		// if no last candle was found, the last candle remains whatever s.lastCandle was
		return s.lastCandle
	}
	for {
		select {
		case <-ctx.Done():
			return
		case subscriptionMsg := <-s.subscriptionMsgCh:
			s.mu.Lock()
			s.handleSubscription(subscriptionMsg)
			s.mu.Unlock()
		case now := <-ticker.C:
			s.mu.RLock()
			s.lastCandle = updateCandles(now)
			s.mu.RUnlock()
		}
	}
}

func (s *CandleUpdates) handleSubscription(subscription subscriptionMsg) {
	if subscription.subscribe {
		s.addSubscription(subscription)
		return
	}
	s.removeSubscription(subscription.id)
}

func (s *CandleUpdates) addSubscription(subscription subscriptionMsg) {
	if s.lastCandle == nil {
		s.subs[subscription.id] = subscription.out
		return
	}
	newSub := map[string]chan entities.Candle{
		subscription.id: subscription.out,
	}
	if rm := s.sendCandlesToSubscribers([]entities.Candle{*s.lastCandle}, newSub); len(rm) == 0 {
		s.subs[subscription.id] = subscription.out
	}
}

func (s *CandleUpdates) removeSubscription(id string) {
	// no lock acquired, the map HAS to be locked when this function is called.
	if ch, ok := s.subs[id]; ok {
		close(ch)
		delete(s.subs, id)
	}
}

func (s *CandleUpdates) closeAllSubscriptions() {
	s.mu.Lock()
	s.lastCandle = nil
	for _, subscriber := range s.subs {
		close(subscriber)
	}
	s.mu.Unlock()
}

// Subscribe returns a unique subscription id and channel on which updates will be sent.
func (s *CandleUpdates) Subscribe() (string, <-chan entities.Candle, error) {
	out := make(chan entities.Candle, s.config.CandleUpdatesStreamBufferSize)

	nextID := s.nextSubscriptionID.Add(1)
	id := fmt.Sprintf("%s-%d", s.candleID, nextID)
	var err error

	if s.config.CandleUpdatesStreamSubscriptionMsgBufferSize == 0 {
		// immediately add, acquire the lock and add to the map.
		s.mu.Lock()
		defer s.mu.Unlock()
		s.subs[id] = out
		// we have some data to send, then try this immediately
		if s.lastCandle != nil {
			newSub := map[string]chan entities.Candle{
				id: out,
			}
			// try to send the last candle to the new subscriber, this will remove the last sub
			// and close the channel if the send fails.
			if rm := s.sendCandlesToSubscribers([]entities.Candle{*s.lastCandle}, newSub); len(rm) != 0 {
				// if rm is not empty, the new subscriber was removed, and the channel was closed.
				return "", nil, ErrNewSubscriberNotReady
			}
		}
		return id, out, nil
	}
	msg := subscriptionMsg{
		subscribe: true,
		id:        id,
		out:       out,
	}

	err = s.sendSubscriptionMessage(msg)
	if err != nil {
		return "", nil, err
	}

	return id, out, nil
}

func (s *CandleUpdates) Unsubscribe(id string) error {
	if s.config.CandleUpdatesStreamSubscriptionMsgBufferSize == 0 {
		// instantly unsubscribe, acquire the lock and remove from the map
		s.mu.Lock()
		if ch, ok := s.subs[id]; ok {
			close(ch)
			delete(s.subs, id)
		}
		s.mu.Unlock()
		return nil
	}
	msg := subscriptionMsg{
		subscribe: false,
		id:        id,
	}

	return s.sendSubscriptionMessage(msg)
}

func (s *CandleUpdates) sendSubscriptionMessage(msg subscriptionMsg) error {
	select {
	case s.subscriptionMsgCh <- msg:
		return nil
	default:
		return fmt.Errorf("failed to send subscription message \"%s\", subscription message buffer is full, try again later", msg)
	}
}

func (s *CandleUpdates) getCandleUpdates(ctx context.Context, now time.Time) ([]entities.Candle, error) {
	ctx, cancelFn := context.WithTimeout(ctx, s.config.CandlesFetchTimeout.Duration)
	defer cancelFn()

	var updates []entities.Candle
	var err error
	if s.lastCandle != nil {
		start := s.lastCandle.PeriodStart
		var candles []entities.Candle
		candles, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleID, &start, &now, entities.CursorPagination{})

		if err != nil {
			return nil, fmt.Errorf("getting candle updates:%w", err)
		}

		// allocate slice rather than doubling cap as we go.
		updates = make([]entities.Candle, 0, len(candles)+1)
		for _, candle := range candles {
			// last candle or newer should be considered an update.
			if !candle.PeriodStart.Before(s.lastCandle.PeriodStart) {
				updates = append(updates, candle)
			}
		}
		return updates, nil
	}
	updates, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleID, nil, &now, entities.CursorPagination{})

	if err != nil {
		return nil, fmt.Errorf("getting candle updates:%w", err)
	}

	return updates, nil
}

func (s *CandleUpdates) sendCandlesToSubscribers(candles []entities.Candle, subscriptions map[string]chan entities.Candle) []string {
	rm := make([]string, 0, len(subscriptions))
	for id, outCh := range subscriptions {
	loop:
		for _, candle := range candles {
			select {
			case outCh <- candle:
			default:
				rm = append(rm, id)
				s.removeSubscription(id)
				break loop
			}
		}
	}
	return rm
}
