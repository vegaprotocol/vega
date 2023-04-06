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

package sqlstore

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/crypto"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/shopspring/decimal"

	"code.vegaprotocol.io/vega/datanode/entities"
	"github.com/georgysavva/scany/pgxscan"
)

const (
	sourceDataTableName    = "trades"
	candlesViewNamePrePend = sourceDataTableName + "_candle_"
)

var candleOrdering = TableOrdering{
	ColumnOrdering{Name: "period_start", Sorting: ASC},
}

type Candles struct {
	*ConnectionSource

	config candlesv2.CandleStoreConfig
	ctx    context.Context
}

type ErrInvalidCandleID struct {
	err error
}

func (e ErrInvalidCandleID) Error() string {
	return e.err.Error()
}

func newErrInvalidCandleID(err error) ErrInvalidCandleID {
	return ErrInvalidCandleID{err: err}
}

func NewCandles(ctx context.Context, connectionSource *ConnectionSource, config candlesv2.CandleStoreConfig) *Candles {
	return &Candles{
		ConnectionSource: connectionSource,
		ctx:              ctx,
		config:           config,
	}
}

// GetCandleDataForTimeSpan gets the candles for a given interval, from and to are optional.
func (cs *Candles) GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
	p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error,
) {
	pageInfo := entities.PageInfo{}

	descriptor, err := candleDescriptorFromCandleID(candleID)
	if err != nil {
		return nil, pageInfo, newErrInvalidCandleID(fmt.Errorf("getting candle data for time span: %w", err))
	}

	exists, err := cs.CandleExists(ctx, descriptor.id)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("getting candles for time span:%w", err)
	}

	if !exists {
		return nil, pageInfo, fmt.Errorf("no candle exists for candle id:%s", candleID)
	}

	var candles []entities.Candle

	query := fmt.Sprintf("SELECT period_start, open, close, high, low, volume, last_update_in_period FROM %s WHERE market_id = $1",
		descriptor.view)

	marketAsBytes, err := hex.DecodeString(descriptor.market)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("invalid market:%w", err)
	}

	args := []interface{}{marketAsBytes}

	if from != nil {
		query = fmt.Sprintf("%s AND period_start >= %s", query, nextBindVar(&args, from))
	}

	if to != nil {
		query = fmt.Sprintf("%s AND period_start < %s", query, nextBindVar(&args, to))
	}

	query, args, err = PaginateQuery[entities.CandleCursor](query, args, candleOrdering, p)
	if err != nil {
		return nil, pageInfo, err
	}

	defer metrics.StartSQLQuery("Candles", "GetCandleDataForTimeSpan")()
	err = pgxscan.Select(ctx, cs.Connection, &candles, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying candles: %w", err)
	}

	var pagedCandles []entities.Candle

	pagedCandles, pageInfo = entities.PageEntities[*v2.CandleEdge](candles, p)

	return pagedCandles, pageInfo, nil
}

// GetCandlesForMarket returns a map of existing intervals to candle ids for the given market.
func (cs *Candles) GetCandlesForMarket(ctx context.Context, market string) (map[string]string, error) {
	intervalToView, err := cs.getIntervalToView(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting existing candles:%w", err)
	}

	candles := map[string]string{}
	for interval := range intervalToView {
		candles[interval] = candleDescriptorFromIntervalAndMarket(interval, market).id
	}
	return candles, nil
}

func (cs *Candles) GetCandleIDForIntervalAndMarket(ctx context.Context, interval string, market string) (bool, string, error) {
	interval, err := cs.normaliseInterval(ctx, interval)
	if err != nil {
		return false, "", fmt.Errorf("invalid interval: %w", err)
	}

	viewAlreadyExists, existingInterval, err := cs.viewExistsForInterval(ctx, interval)
	if err != nil {
		return false, "", fmt.Errorf("checking for existing view: %w", err)
	}

	if viewAlreadyExists {
		descriptor := candleDescriptorFromIntervalAndMarket(existingInterval, market)
		return true, descriptor.id, nil
	}

	return false, "", nil
}

func (cs *Candles) getIntervalToView(ctx context.Context) (map[string]string, error) {
	query := fmt.Sprintf("SELECT table_name AS view_name FROM INFORMATION_SCHEMA.views WHERE table_name LIKE '%s%%'",
		candlesViewNamePrePend)
	defer metrics.StartSQLQuery("Candles", "GetIntervalToView")()
	rows, err := cs.Connection.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fetching existing views for interval: %w", err)
	}

	var viewNames []string
	for rows.Next() {
		var viewName string
		err := rows.Scan(&viewName)
		if err != nil {
			return nil, fmt.Errorf("fetching existing views for interval: %w", err)
		}
		viewNames = append(viewNames, viewName)
	}

	result := map[string]string{}
	for _, viewName := range viewNames {
		interval, err := getIntervalFromViewName(viewName)
		if err != nil {
			return nil, fmt.Errorf("fetching existing views for interval: %w", err)
		}

		result[interval] = viewName
	}
	return result, nil
}

func (cs *Candles) CandleExists(ctx context.Context, candleID string) (bool, error) {
	descriptor, err := candleDescriptorFromCandleID(candleID)
	if err != nil {
		return false, fmt.Errorf("candle exists:%w", err)
	}

	exists, _, err := cs.viewExistsForInterval(ctx, descriptor.interval)
	if err != nil {
		return false, fmt.Errorf("candle exists:%w", err)
	}

	return exists, nil
}

func (cs *Candles) viewExistsForInterval(ctx context.Context, interval string) (bool, string, error) {
	intervalToView, err := cs.getIntervalToView(ctx)
	if err != nil {
		return false, "", fmt.Errorf("checking if view exist for interval:%w", err)
	}

	if _, ok := intervalToView[interval]; ok {
		return true, interval, nil
	}

	// Also check for existing Intervals that are specified differently but amount to the same thing  (i.e 7 days = 1 week)
	existingIntervals := map[int64]string{}
	for existingInterval := range intervalToView {
		if existingInterval == "block" {
			continue
		}
		seconds, err := cs.getIntervalSeconds(ctx, existingInterval)
		if err != nil {
			return false, "", fmt.Errorf("checking if view exists for interval:%w", err)
		}
		existingIntervals[seconds] = existingInterval
	}

	seconds, err := cs.getIntervalSeconds(ctx, interval)
	if err != nil {
		return false, "", fmt.Errorf("checking if view exists for interval:%w", err)
	}

	if existingInterval, ok := existingIntervals[seconds]; ok {
		return true, existingInterval, nil
	}

	return false, "", nil
}

func (cs *Candles) normaliseInterval(ctx context.Context, interval string) (string, error) {
	var normalizedInterval string

	defer metrics.StartSQLQuery("Candles", "normaliseInterval")()
	_, err := cs.Connection.Exec(ctx, "SET intervalstyle = 'postgres_verbose' ")
	if err != nil {
		return "", fmt.Errorf("normalising interval, failed to set interval style:%w", err)
	}

	query := fmt.Sprintf("select cast( INTERVAL '%s' as text)", interval)
	row := cs.Connection.QueryRow(ctx, query)

	err = row.Scan(&normalizedInterval)
	if err != nil {
		return "", fmt.Errorf("normalising interval:%s :%w", interval, err)
	}

	normalizedInterval = strings.ReplaceAll(normalizedInterval, "@ ", "")

	return normalizedInterval, nil
}

func (cs *Candles) getIntervalSeconds(ctx context.Context, interval string) (int64, error) {
	var seconds decimal.Decimal

	defer metrics.StartSQLQuery("Candles", "getIntervalSeconds")()
	query := fmt.Sprintf("SELECT EXTRACT(epoch FROM INTERVAL '%s')", interval)
	row := cs.Connection.QueryRow(ctx, query)

	err := row.Scan(&seconds)
	if err != nil {
		return 0, err
	}

	return seconds.IntPart(), nil
}

func getIntervalFromViewName(viewName string) (string, error) {
	split := strings.Split(viewName, candlesViewNamePrePend)
	if len(split) != 2 {
		return "", fmt.Errorf("view name has unexpected format:%s", viewName)
	}
	return strings.ReplaceAll(split[1], "_", " "), nil
}

func getViewNameForInterval(interval string) string {
	return candlesViewNamePrePend + strings.ReplaceAll(interval, " ", "_")
}

type candleDescriptor struct {
	id       string
	view     string
	interval string
	market   string
}

func candleDescriptorFromCandleID(id string) (candleDescriptor, error) {
	idx := strings.LastIndex(id, "_")

	if idx == -1 {
		return candleDescriptor{}, fmt.Errorf("invalid candle id:%s", id)
	}

	market := id[idx+1:]
	view := id[:idx]

	split := strings.Split(view, candlesViewNamePrePend)
	if len(split) != 2 {
		return candleDescriptor{}, fmt.Errorf("parsing candle id, view name has unexpected format:%s", id)
	}

	interval, err := getIntervalFromViewName(view)
	if err != nil {
		return candleDescriptor{}, fmt.Errorf("parsing candleDescriptor id, failed to get interval from view name:%w", err)
	}

	if !crypto.IsValidVegaID(market) {
		return candleDescriptor{}, fmt.Errorf("not a valid market id: %v", market)
	}

	return candleDescriptor{
		id:       id,
		view:     view,
		interval: interval,
		market:   market,
	}, nil
}

func candleDescriptorFromIntervalAndMarket(interval string, market string) candleDescriptor {
	view := getViewNameForInterval(interval)
	id := view + "_" + market

	return candleDescriptor{
		id:       id,
		view:     view,
		interval: interval,
		market:   market,
	}
}
