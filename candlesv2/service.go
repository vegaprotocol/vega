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
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
)

// CandleStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/data-node/candlesv2 CandleStore
type CandleStore interface {
	GetCandleDataForTimeSpan(ctx context.Context, candleId string, from *time.Time, to *time.Time,
		p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error)
	GetCandlesForMarket(ctx context.Context, market string) (map[string]string, error)
	CandleExists(ctx context.Context, candleId string) (bool, error)
	GetCandleIdForIntervalAndMarket(ctx context.Context, interval string, market string) (bool, string, error)
}

type Svc struct {
	Config
	CandleStore
	ctx context.Context
	log *logging.Logger

	candleIdToUpdatesStream  map[string]*candleUpdates
	subscriptionIdToCandleId map[string]string
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
		candleIdToUpdatesStream:  map[string]*candleUpdates{},
		subscriptionIdToCandleId: map[string]string{},
	}
}

// Subscribe to a channel of new or updated candles. The subscriber id will be returned as an uint64 value
// and must be retained for future reference and to Unsubscribe.
func (cs *Svc) Subscribe(ctx context.Context, candleId string) (string, <-chan entities.Candle, error) {
	cs.updatesSubscriptionMutex.Lock()
	defer cs.updatesSubscriptionMutex.Unlock()

	exists, err := cs.CandleExists(ctx, candleId)
	if err != nil {
		return "", nil, fmt.Errorf("subscribing to candles:%w", err)
	}

	if !exists {
		return "", nil, fmt.Errorf("no candle exists for candle id:%s", candleId)
	}

	if _, ok := cs.candleIdToUpdatesStream[candleId]; !ok {
		updates, err := NewCandleUpdates(cs.ctx, cs.log, candleId, cs, cs.Config.CandleUpdates)
		if err != nil {
			return "", nil, fmt.Errorf("subsribing to candle updates:%w", err)
		}

		cs.candleIdToUpdatesStream[candleId] = updates
	}

	updatesStream := cs.candleIdToUpdatesStream[candleId]
	subscriptionId, out := updatesStream.Subscribe()
	cs.subscriptionIdToCandleId[subscriptionId] = candleId

	return subscriptionId, out, nil
}

func (cs *Svc) Unsubscribe(subscriptionId string) error {
	cs.updatesSubscriptionMutex.Lock()
	defer cs.updatesSubscriptionMutex.Unlock()

	if candleId, ok := cs.subscriptionIdToCandleId[subscriptionId]; ok {
		updatesStream := cs.candleIdToUpdatesStream[candleId]
		updatesStream.Unsubscribe(subscriptionId)
		return nil
	} else {
		return fmt.Errorf("no subscription found for id %s", subscriptionId)
	}
}
