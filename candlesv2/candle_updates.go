// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package candlesv2

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
)

type candleSource interface {
	GetCandleDataForTimeSpan(ctx context.Context, candleId string, from *time.Time, to *time.Time,
		p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error)
}

type subscribeRequest struct {
	id  string
	out chan entities.Candle
}

type candleUpdates struct {
	log                *logging.Logger
	candleSource       candleSource
	candleId           string
	subscribeChan      chan subscribeRequest
	unsubscribeChan    chan string
	nextSubscriptionId uint64
	config             CandleUpdatesConfig
}

func NewCandleUpdates(ctx context.Context, log *logging.Logger, candleId string, candleSource candleSource,
	config CandleUpdatesConfig) (*candleUpdates, error,
) {
	ces := &candleUpdates{
		log:             log,
		candleSource:    candleSource,
		candleId:        candleId,
		config:          config,
		subscribeChan:   make(chan subscribeRequest),
		unsubscribeChan: make(chan string),
	}

	go ces.run(ctx)

	return ces, nil
}

func (s *candleUpdates) run(ctx context.Context) {
	subscriptions := map[string]chan entities.Candle{}
	defer closeAllSubscriptions(subscriptions)

	ticker := time.NewTicker(s.config.CandleUpdatesStreamInterval.Duration)
	var lastCandle *entities.Candle

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(subscriptions) > 0 {
				candles, err := s.getCandleUpdates(ctx, lastCandle)
				if err != nil {
					s.log.Errorf("failed to get candles, closing stream for candle id %s: %w", s.candleId, err)
					return
				}

				if len(candles) > 0 {
					lastCandle = &candles[len(candles)-1]
				}

				s.sendCandles(candles, subscriptions)

			} else {
				lastCandle = nil
			}
		case subscription := <-s.subscribeChan:
			subscriptions[subscription.id] = subscription.out
			if lastCandle != nil {
				s.sendCandles([]entities.Candle{*lastCandle}, map[string]chan entities.Candle{subscription.id: subscription.out})
			}
		case id := <-s.unsubscribeChan:
			removeSubscription(subscriptions, id)
		}
	}
}

func removeSubscription(subscriptions map[string]chan entities.Candle, subscriptionId string) {
	if _, ok := subscriptions[subscriptionId]; ok {
		close(subscriptions[subscriptionId])
		delete(subscriptions, subscriptionId)
	}
}

func closeAllSubscriptions(subscribers map[string]chan entities.Candle) {
	for _, subscriber := range subscribers {
		close(subscriber)
	}
}

// Subscribe returns a unique subscription id and channel on which updates will be sent
func (s *candleUpdates) Subscribe() (string, <-chan entities.Candle) {
	out := make(chan entities.Candle, s.config.CandleUpdatesStreamBufferSize)

	nextId := atomic.AddUint64(&s.nextSubscriptionId, 1)
	subscriptionId := fmt.Sprintf("%s-%d", s.candleId, nextId)
	s.subscribeChan <- subscribeRequest{
		id:  subscriptionId,
		out: out,
	}

	return subscriptionId, out
}

func (s *candleUpdates) Unsubscribe(subscriptionId string) {
	s.unsubscribeChan <- subscriptionId
}

func (s *candleUpdates) getCandleUpdates(ctx context.Context, lastCandle *entities.Candle) ([]entities.Candle, error) {
	ctx, cancelFn := context.WithTimeout(ctx, s.config.CandlesFetchTimeout.Duration)
	defer cancelFn()

	var updates []entities.Candle
	var err error
	if lastCandle != nil {
		start := lastCandle.PeriodStart
		var candles []entities.Candle
		candles, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleId, &start, nil, entities.CursorPagination{})

		if err != nil {
			return nil, fmt.Errorf("getting candle updates:%w", err)
		}

		for _, candle := range candles {
			if candle.LastUpdateInPeriod.After(lastCandle.LastUpdateInPeriod) {
				updates = append(updates, candle)
			}
		}
	} else {
		last := int32(1)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil)
		if err != nil {
			return nil, err
		}
		updates, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleId, nil, nil, pagination)

		if err != nil {
			return nil, fmt.Errorf("getting candle updates:%w", err)
		}
	}

	return updates, nil
}

func (s *candleUpdates) sendCandles(candles []entities.Candle, subscriptions map[string]chan entities.Candle) {
	var slowConsumers []string

	for subscriptionId, outCh := range subscriptions {
		for _, candle := range candles {
			if len(outCh) < cap(outCh) {
				outCh <- candle
			} else {
				slowConsumers = append(slowConsumers, subscriptionId)
				break
			}
		}
	}

	for _, slowConsumerId := range slowConsumers {
		s.log.Warningf("slow consumer detected, removing subscription %s", slowConsumerId)
		removeSubscription(subscriptions, slowConsumerId)
	}
}
