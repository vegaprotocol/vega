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
	"fmt"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

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
	log                 *logging.Logger
	candleSource        candleSource
	candleID            string
	subscriptionMsgChan chan subscriptionMsg
	nextSubscriptionID  atomic.Uint64
	config              CandleUpdatesConfig
}

func NewCandleUpdates(ctx context.Context, log *logging.Logger, candleID string, candleSource candleSource,
	config CandleUpdatesConfig,
) *CandleUpdates {
	ces := &CandleUpdates{
		log:                 log,
		candleSource:        candleSource,
		candleID:            candleID,
		config:              config,
		subscriptionMsgChan: make(chan subscriptionMsg, config.CandleUpdatesStreamSubscriptionMsgBufferSize),
	}

	go ces.run(ctx)

	return ces
}

func (s *CandleUpdates) run(ctx context.Context) {
	subscriptions := map[string]chan entities.Candle{}
	defer closeAllSubscriptions(subscriptions)

	ticker := time.NewTicker(s.config.CandleUpdatesStreamInterval.Duration)
	defer ticker.Stop()
	var lastCandle *entities.Candle

	errorGettingCandleUpdates := false
	for {
		select {
		case <-ctx.Done():
			return
		case subscriptionMsg := <-s.subscriptionMsgChan:
			s.handleSubscription(subscriptions, subscriptionMsg, lastCandle)
		case now := <-ticker.C:
			if len(subscriptions) == 0 {
				lastCandle = nil
				continue
			}
			candles, err := s.getCandleUpdates(ctx, lastCandle, now)
			if err != nil {
				if !errorGettingCandleUpdates {
					s.log.Errorf("failed to get candles for candle id", logging.String("candle", s.candleID), logging.Error(err))
				}
				errorGettingCandleUpdates = true
				continue
			}
			if errorGettingCandleUpdates {
				s.log.Infof("Successfully got candles for candle", logging.String("candle", s.candleID))
				errorGettingCandleUpdates = false
			}
			if len(candles) > 0 {
				lastCandle = &candles[len(candles)-1]
			}

			s.sendCandlesToSubscribers(candles, subscriptions)
		}
	}
}

func (s *CandleUpdates) handleSubscription(subscriptions map[string]chan entities.Candle, subscription subscriptionMsg, lastCandle *entities.Candle) {
	if subscription.subscribe {
		s.addSubscription(subscriptions, subscription, lastCandle)
	} else {
		removeSubscription(subscriptions, subscription.id)
	}
}

func (s *CandleUpdates) addSubscription(subscriptions map[string]chan entities.Candle, subscription subscriptionMsg, lastCandle *entities.Candle) {
	subscriptions[subscription.id] = subscription.out
	if lastCandle != nil {
		s.sendCandlesToSubscribers([]entities.Candle{*lastCandle}, map[string]chan entities.Candle{subscription.id: subscription.out})
	}
}

func removeSubscription(subscriptions map[string]chan entities.Candle, subscriptionID string) {
	if _, ok := subscriptions[subscriptionID]; ok {
		close(subscriptions[subscriptionID])
		delete(subscriptions, subscriptionID)
	}
}

func closeAllSubscriptions(subscribers map[string]chan entities.Candle) {
	for _, subscriber := range subscribers {
		close(subscriber)
	}
}

// Subscribe returns a unique subscription id and channel on which updates will be sent.
func (s *CandleUpdates) Subscribe() (string, <-chan entities.Candle, error) {
	out := make(chan entities.Candle, s.config.CandleUpdatesStreamBufferSize)

	nextID := s.nextSubscriptionID.Add(1)
	subscriptionID := fmt.Sprintf("%s-%d", s.candleID, nextID)

	msg := subscriptionMsg{
		subscribe: true,
		id:        subscriptionID,
		out:       out,
	}

	err := s.sendSubscriptionMessage(msg)
	if err != nil {
		return "", nil, err
	}

	return subscriptionID, out, nil
}

func (s *CandleUpdates) Unsubscribe(subscriptionID string) error {
	msg := subscriptionMsg{
		subscribe: false,
		id:        subscriptionID,
	}

	return s.sendSubscriptionMessage(msg)
}

func (s *CandleUpdates) sendSubscriptionMessage(msg subscriptionMsg) error {
	if s.config.CandleUpdatesStreamSubscriptionMsgBufferSize == 0 {
		s.subscriptionMsgChan <- msg
	} else {
		select {
		case s.subscriptionMsgChan <- msg:
		default:
			return fmt.Errorf("failed to send subscription message \"%s\", subscription message buffer is full, try again later", msg)
		}
	}
	return nil
}

func (s *CandleUpdates) getCandleUpdates(ctx context.Context, lastCandle *entities.Candle, now time.Time) ([]entities.Candle, error) {
	ctx, cancelFn := context.WithTimeout(ctx, s.config.CandlesFetchTimeout.Duration)
	defer cancelFn()

	var updates []entities.Candle
	var err error
	if lastCandle != nil {
		start := lastCandle.PeriodStart
		var candles []entities.Candle
		candles, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleID, &start, &now, entities.CursorPagination{})

		if err != nil {
			return nil, fmt.Errorf("getting candle updates:%w", err)
		}

		for _, candle := range candles {
			if candle.PeriodStart.After(lastCandle.PeriodStart) {
				updates = append(updates, candle)
			}
		}
	} else {
		last := int32(1)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		if err != nil {
			return nil, err
		}
		updates, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleID, nil, &now, pagination)

		if err != nil {
			return nil, fmt.Errorf("getting candle updates:%w", err)
		}
	}

	return updates, nil
}

func (s *CandleUpdates) sendCandlesToSubscribers(candles []entities.Candle, subscriptions map[string]chan entities.Candle) {
	for subscriptionID, outCh := range subscriptions {
		for _, candle := range candles {
			select {
			case outCh <- candle:
			default:
				removeSubscription(subscriptions, subscriptionID)
				break
			}
		}
	}
}
