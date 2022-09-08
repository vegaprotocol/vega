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
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

// CandleStore ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/datanode/candlesv2 CandleStore
type CandleStore interface {
	GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
		p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error)
	GetCandlesForMarket(ctx context.Context, market string) (map[string]string, error)
	CandleExists(ctx context.Context, candleID string) (bool, error)
	GetCandleIDForIntervalAndMarket(ctx context.Context, interval string, market string) (bool, string, error)
}

type Svc struct {
	Config
	CandleStore
	ctx context.Context
	log *logging.Logger

	candleIDToUpdatesStream  map[string]*CandleUpdates
	updatesSubscriptionMutex sync.Mutex
}

func NewService(ctx context.Context, log *logging.Logger, config Config, candleStore CandleStore) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		ctx:                     ctx,
		log:                     log,
		Config:                  config,
		CandleStore:             candleStore,
		candleIDToUpdatesStream: map[string]*CandleUpdates{},
	}
}

// Subscribe to a channel of new or updated candles. The subscriber id will be returned as an uint64 value
// and must be retained for future reference and to Unsubscribe.
func (cs *Svc) Subscribe(ctx context.Context, candleID string) (uint64, <-chan entities.Candle, error) {
	cs.updatesSubscriptionMutex.Lock()
	defer cs.updatesSubscriptionMutex.Unlock()

	exists, err := cs.CandleExists(ctx, candleID)
	if err != nil {
		return 0, nil, fmt.Errorf("subscribing to candles:%w", err)
	}

	if !exists {
		return 0, nil, fmt.Errorf("no candle exists for candle id:%s", candleID)
	}

	updatesStream, ok := cs.candleIDToUpdatesStream[candleID]

	// If we don't have a stream for this candle, or existing update stream has stopped running,
	// (for example it errored, or quit because there were no remaining subscribers) -
	// Then make a new one. There is a very small chance that the stream might stop in-between
	// us checking here an subscribing below; there's no race there - the subscription will just
	// fail with an `ErrCandleUpdatesStopped`
	if !ok || (ok && !updatesStream.IsRunning()) {
		updatesStream, err = NewCandleUpdates(cs.ctx, cs.log, candleID, cs, cs.Config.CandleUpdates)
		if err != nil {
			return 0, nil, fmt.Errorf("subscribing to candle updates:%w", err)
		}
		cs.candleIDToUpdatesStream[candleID] = updatesStream
	}

	out, subscriptionID, err := updatesStream.Subscribe()
	if err != nil {
		return 0, nil, fmt.Errorf("subscribing to candle updates:%w", err)
	}
	return subscriptionID, out, nil
}

func (cs *Svc) Unsubscribe(candleID string, subscriptionID uint64) error {
	cs.updatesSubscriptionMutex.Lock()
	defer cs.updatesSubscriptionMutex.Unlock()

	if updatesStream, ok := cs.candleIDToUpdatesStream[candleID]; ok {
		return updatesStream.Unsubscribe(subscriptionID)
	}
	return fmt.Errorf("no subscription found for id %s/%v", candleID, subscriptionID)
}
