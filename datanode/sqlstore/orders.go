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
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

const (
	sqlOrderColumns = `id, market_id, party_id, side, price,
                       size, remaining, time_in_force, type, status,
                       reference, reason, version, batch_id, pegged_offset,
                       pegged_reference, lp_id, created_at, updated_at, expires_at,
                       tx_hash, vega_time, seq_num`

	ordersFilterDateColumn = "vega_time"
)

type Orders struct {
	*ConnectionSource
	batcher MapBatcher[entities.OrderKey, entities.Order]
}

var ordersOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "seq_num", Sorting: ASC},
}

func NewOrders(connectionSource *ConnectionSource, logger *logging.Logger) *Orders {
	a := &Orders{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.OrderKey, entities.Order](
			"orders",
			entities.OrderColumns),
	}
	return a
}

func (os *Orders) Flush(ctx context.Context) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "Flush")()
	return os.batcher.Flush(ctx, os.Connection)
}

// Add inserts an order update row into the database if an row for this (block time, order id, version)
// does not already exist; otherwise update the existing row with information supplied.
// Currently we only store the last update to an order per block, so the order history is not
// complete if multiple updates happen in one block.
func (os *Orders) Add(o entities.Order) error {
	os.batcher.Add(o)
	return nil
}

// GetAll returns all updates to all orders (including changes to orders that don't increment the version number).
func (os *Orders) GetAll(ctx context.Context) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetAll")()
	orders := []entities.Order{}
	query := fmt.Sprintf("SELECT %s FROM orders", sqlOrderColumns)
	err := pgxscan.Select(ctx, os.Connection, &orders, query)
	return orders, err
}

// GetByOrderId returns the last update of the order with the given ID.
func (os *Orders) GetOrder(ctx context.Context, orderIDStr string, version *int32) (entities.Order, error) {
	var err error
	order := entities.Order{}
	orderID := entities.OrderID(orderIDStr)

	defer metrics.StartSQLQuery("Orders", "GetByOrderID")()
	if version != nil && *version > 0 {
		query := fmt.Sprintf("SELECT %s FROM orders_current_versions WHERE id=$1 and version=$2", sqlOrderColumns)
		err = pgxscan.Get(ctx, os.Connection, &order, query, orderID, version)
	} else {
		query := fmt.Sprintf("SELECT %s FROM orders_current WHERE id=$1", sqlOrderColumns)
		err = pgxscan.Get(ctx, os.Connection, &order, query, orderID)
	}
	return order, err
}

// GetByMarket returns the last update of the all the orders in a particular market.
func (os *Orders) GetByMarket(ctx context.Context, marketIDStr string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByMarket")()
	marketID := entities.MarketID(marketIDStr)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE market_id=$1`, sqlOrderColumns)
	args := []interface{}{marketID}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByParty returns the last update of the all the orders in a particular party.
func (os *Orders) GetByParty(ctx context.Context, partyIDStr string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByParty")()
	partyID := entities.PartyID(partyIDStr)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE party_id=$1`, sqlOrderColumns)
	args := []interface{}{partyID}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByReference returns the last update of orders with the specified user-suppled reference.
func (os *Orders) GetByReference(ctx context.Context, reference string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByReference")()
	query := fmt.Sprintf(`SELECT %s from orders_current WHERE reference=$1`, sqlOrderColumns)
	args := []interface{}{reference}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByReference returns the last update of orders with the specified user-suppled reference.
func (os *Orders) GetByReferencePaged(ctx context.Context, reference string, p entities.CursorPagination) ([]entities.Order, entities.PageInfo, error) {
	return os.ListOrders(ctx, nil, nil, &reference, false, p, entities.DateRange{})
}

// GetAllVersionsByOrderID the last update to all versions (e.g. manual changes that lead to
// incrementing the version field) of a given order id.
func (os *Orders) GetAllVersionsByOrderID(ctx context.Context, id string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetAllVersionsByOrderID")()
	query := fmt.Sprintf(`SELECT %s from orders_current_versions WHERE id=$1`, sqlOrderColumns)
	args := []interface{}{entities.OrderID(id)}
	return os.queryOrders(ctx, query, args, &p)
}

// GetLiveOrders fetches all currently live orders so the market depth data can be rebuilt
// from the orders data in the database.
func (os *Orders) GetLiveOrders(ctx context.Context) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetLiveOrders")()
	query := fmt.Sprintf(`select %s from orders_live
where type = 1
and time_in_force not in (3, 4)
and status in (1, 7)
order by vega_time, seq_num`, sqlOrderColumns)
	return os.queryOrders(ctx, query, nil, nil)
}

// -------------------------------------------- Utility Methods

func (os *Orders) queryOrders(ctx context.Context, query string, args []interface{}, p *entities.OffsetPagination) ([]entities.Order, error) {
	if p != nil {
		query, args = paginateOrderQuery(query, args, *p)
	}

	orders := []entities.Order{}
	err := pgxscan.Select(ctx, os.Connection, &orders, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying orders: %w", err)
	}
	return orders, nil
}

func (os *Orders) queryOrdersWithCursorPagination(ctx context.Context, query string, args []interface{},
	pagination entities.CursorPagination,
) ([]entities.Order, entities.PageInfo, error) {
	var (
		err      error
		orders   []entities.Order
		pageInfo entities.PageInfo
	)

	query, args, err = PaginateQuery[entities.OrderCursor](query, args, ordersOrdering, pagination)
	if err != nil {
		return orders, pageInfo, err
	}
	err = pgxscan.Select(ctx, os.Connection, &orders, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying orders: %w", err)
	}

	orders, pageInfo = entities.PageEntities[*v2.OrderEdge](orders, pagination)
	return orders, pageInfo, nil
}

func paginateOrderQuery(query string, args []interface{}, p entities.OffsetPagination) (string, []interface{}) {
	dir := "ASC"
	if p.Descending {
		dir = "DESC"
	}

	var limit interface{}
	if p.Limit != 0 {
		limit = p.Limit
	}

	query = fmt.Sprintf(" %s ORDER BY vega_time %s, id %s LIMIT %s OFFSET %s",
		query, dir, dir, nextBindVar(&args, limit), nextBindVar(&args, p.Skip))

	return query, args
}

func (os *Orders) ListOrders(ctx context.Context, party *string, market *string, reference *string, liveOnly bool, p entities.CursorPagination,
	dateRange entities.DateRange,
) ([]entities.Order, entities.PageInfo, error) {
	var filters []filter
	if party != nil {
		filters = append(filters, filter{"party_id", entities.PartyID(*party)})
	}

	if market != nil {
		filters = append(filters, filter{"market_id", entities.MarketID(*market)})
	}

	if reference != nil {
		filters = append(filters, filter{"reference", *reference})
	}

	where, args := buildWhereClause(filters...)

	table := "orders_current"
	if liveOnly {
		table = "orders_live"
	}

	query := fmt.Sprintf(`SELECT %s from %s %s`, sqlOrderColumns, table, where)
	query, args = filterDateRange(query, ordersFilterDateColumn, dateRange, args...)

	defer metrics.StartSQLQuery("Orders", "GetByMarketPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, args, p)
}

func (os *Orders) ListOrderVersions(ctx context.Context, orderIDStr string, p entities.CursorPagination) ([]entities.Order, entities.PageInfo, error) {
	if orderIDStr == "" {
		return nil, entities.PageInfo{}, errors.New("orderID is required")
	}
	orderID := entities.OrderID(orderIDStr)
	query := fmt.Sprintf(`SELECT %s from orders_current_versions WHERE id=$1`, sqlOrderColumns)
	defer metrics.StartSQLQuery("Orders", "GetByOrderIDPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, []interface{}{orderID}, p)
}

type filter struct {
	colName string
	value   any
}

func buildWhereClause(filters ...filter) (string, []any) {
	whereBuilder := strings.Builder{}
	var args []any
	filterNum := 0
	for _, filter := range filters {
		if filter.value != nil {
			filterNum++
			if filterNum == 1 {
				whereBuilder.WriteString(fmt.Sprintf("WHERE %s = $1", filter.colName))
				args = append(args, filter.value)
			} else {
				whereBuilder.WriteString(fmt.Sprintf(" AND %s = $%d", filter.colName, filterNum))
				args = append(args, filter.value)
			}
		}
	}

	return whereBuilder.String(), args
}
