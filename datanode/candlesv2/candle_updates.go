// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
)

var ErrCandleUpdatesStopped = errors.New("candle updates stopped")

type candleSource interface {
	GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
		p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error)
}

type CandleUpdates struct {
	log          *logging.Logger
	candleSource candleSource
	candleID     string
	config       CandleUpdatesConfig
	running      atomic.Bool
	lastCandle   atomic.Pointer[entities.Candle]
	notifier     utils.Notifier[entities.Candle]
	stopping     sync.Mutex
}

func NewCandleUpdates(ctx context.Context, log *logging.Logger, candleID string, candleSource candleSource,
	config CandleUpdatesConfig) (*CandleUpdates, error,
) {
	observerName := fmt.Sprintf("candles(%s)", candleID)
	ces := &CandleUpdates{
		log:          log,
		candleSource: candleSource,
		candleID:     candleID,
		config:       config,
		notifier:     utils.NewNotifier[entities.Candle](observerName, log, 10),
	}

	ces.running.Store(true)
	go ces.run(ctx)

	return ces, nil
}

func (s *CandleUpdates) IsRunning() bool {
	return s.running.Load()
}

func (s *CandleUpdates) run(ctx context.Context) {
	// This is a little bit subtle: We need to ensure that once this goroutine exits, there are
	// no active subscriptions on our observer, so we take a mutex in s.stop() that prevents
	// the (unlikely) case that Subscribe() is called while we are shutting down; or else we will
	// return a channel that will never be serviced.
	defer s.stop()

	ticker := time.NewTicker(s.config.CandleUpdatesStreamInterval.Duration)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.notifier.GetSubscribersCount() > 0 {
				candles, err := s.getCandleUpdates(ctx, s.lastCandle.Load())
				if err != nil {
					s.log.Errorf("failed to get candles, closing stream for candle id %s: %w", s.candleID, err)
					return
				}

				if len(candles) > 0 {
					s.lastCandle.Store(&candles[len(candles)-1])
				}

				for _, candle := range candles {
					s.notifier.Notify(candle)
				}
			} else {
				s.lastCandle.Store(nil)
			}
		}
	}
}

func (s *CandleUpdates) stop() {
	s.stopping.Lock()
	defer s.stopping.Unlock()
	s.notifier.UnsubscribeAll()
	s.running.Store(false)
}

func (s *CandleUpdates) Subscribe() (<-chan entities.Candle, uint64, error) {
	s.stopping.Lock()
	defer s.stopping.Unlock()

	if !s.IsRunning() {
		return nil, 0, ErrCandleUpdatesStopped
	}

	if lastCandle := s.lastCandle.Load(); lastCandle != nil {
		return s.notifier.SubscribeAndNotify(*lastCandle)
	}

	ch, id := s.notifier.Subscribe()
	return ch, id, nil
}

func (s *CandleUpdates) Unsubscribe(subscriptionID uint64) error {
	// If we're not running, will already have been unsubscribed
	if !s.IsRunning() {
		return nil
	}

	return s.notifier.Unsubscribe(subscriptionID)
}

func (s *CandleUpdates) getCandleUpdates(ctx context.Context, lastCandle *entities.Candle) ([]entities.Candle, error) {
	ctx, cancelFn := context.WithTimeout(ctx, s.config.CandlesFetchTimeout.Duration)
	defer cancelFn()

	var updates []entities.Candle
	var err error
	if lastCandle != nil {
		start := lastCandle.PeriodStart
		var candles []entities.Candle
		candles, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleID, &start, nil, entities.CursorPagination{})

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
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		if err != nil {
			return nil, err
		}
		updates, _, err = s.candleSource.GetCandleDataForTimeSpan(ctx, s.candleID, nil, nil, pagination)

		if err != nil {
			return nil, fmt.Errorf("getting candle updates:%w", err)
		}
	}

	return updates, nil
}
