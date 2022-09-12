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
	subscriptionIDToCandleID map[string]string
	updatesSubscriptionMutex sync.Mutex
}

func NewService(ctx context.Context, log *logging.Logger, config Config, candleStore CandleStore) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		ctx:                      ctx,
		log:                      log,
		Config:                   config,
		CandleStore:              candleStore,
		candleIDToUpdatesStream:  map[string]*CandleUpdates{},
		subscriptionIDToCandleID: map[string]string{},
	}
}

// Subscribe to a channel of new or updated candles. The subscriber id will must be retained for future reference and to Unsubscribe.
func (cs *Svc) Subscribe(ctx context.Context, candleID string) (string, <-chan entities.Candle, error) {
	cs.updatesSubscriptionMutex.Lock()
	defer cs.updatesSubscriptionMutex.Unlock()

	exists, err := cs.CandleExists(ctx, candleID)
	if err != nil {
		return "", nil, fmt.Errorf("subscribing to candles:%w", err)
	}

	if !exists {
		return "", nil, fmt.Errorf("no candle exists for candle id:%s", candleID)
	}

	if _, ok := cs.candleIDToUpdatesStream[candleID]; !ok {
		updates := NewCandleUpdates(cs.ctx, cs.log, candleID, cs, cs.Config.CandleUpdates)
		cs.candleIDToUpdatesStream[candleID] = updates
	}

	updatesStream := cs.candleIDToUpdatesStream[candleID]
	subscriptionID, out, err := updatesStream.Subscribe()
	if err != nil {
		return "", nil, fmt.Errorf("failed to subscribe to candle %s: %w", candleID, err)
	}

	cs.subscriptionIDToCandleID[subscriptionID] = candleID

	return subscriptionID, out, nil
}

func (cs *Svc) Unsubscribe(subscriptionID string) error {
	cs.updatesSubscriptionMutex.Lock()
	defer cs.updatesSubscriptionMutex.Unlock()

	if candleID, ok := cs.subscriptionIDToCandleID[subscriptionID]; ok {
		updatesStream := cs.candleIDToUpdatesStream[candleID]
		err := updatesStream.Unsubscribe(subscriptionID)
		if err != nil {
			return fmt.Errorf("failed to unsubscribe from candle %s: %w", candleID, err)
		}
		return nil
	}
	return fmt.Errorf("no subscription found for id %s", subscriptionID)
}
