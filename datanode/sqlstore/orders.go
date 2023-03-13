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

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

const (
	sqlOrderColumns = `id, market_id, party_id, side, price,
                       size, remaining, time_in_force, type, status,
                       reference, reason, version, batch_id, pegged_offset,
                       pegged_reference, lp_id, created_at, updated_at, expires_at,
                       tx_hash, vega_time, seq_num, post_only, reduce_only`

	ordersFilterDateColumn = "vega_time"

	OrdersTableName = "orders"
)

type Orders struct {
	*ConnectionSource
	batcher MapBatcher[entities.OrderKey, entities.Order]
}

var ordersOrdering = TableOrdering{
	ColumnOrdering{Name: "created_at", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: DESC},
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

func NewOrders(connectionSource *ConnectionSource) *Orders {
	a := &Orders{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.OrderKey, entities.Order](
			OrdersTableName,
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

// GetOrder returns the last update of the order with the given ID.
func (os *Orders) GetOrder(ctx context.Context, orderIDStr string, version *int32) (entities.Order, error) {
	var err error
	order := entities.Order{}
	orderID := entities.OrderID(orderIDStr)

	defer metrics.StartSQLQuery("Orders", "GetByOrderID")()
	if version != nil && *version > 0 {
		query := fmt.Sprintf("SELECT %s FROM orders_current_versions WHERE id=$1 and version=$2", sqlOrderColumns)
		err = pgxscan.Get(ctx, os.Connection, &order, query, orderID, version)
	} else {
		query := fmt.Sprintf("SELECT %s FROM orders_current_desc WHERE id=$1", sqlOrderColumns)
		err = pgxscan.Get(ctx, os.Connection, &order, query, orderID)
	}

	return order, os.wrapE(err)
}

// GetByMarketAndID returns all orders with given IDs for a market.
func (os *Orders) GetByMarketAndID(ctx context.Context, marketIDstr string, orderIDs []string) ([]entities.Order, error) {
	if len(orderIDs) == 0 {
		os.log.Warn("GetByMarketAndID called with an empty order slice",
			logging.String("market ID", marketIDstr),
		)
		return nil, nil
	}
	defer metrics.StartSQLQuery("Orders", "GetByMarketAndID")()
	marketID := entities.MarketID(marketIDstr)
	// IDs := make([]entities.OrderID, 0, len(orderIDs))
	IDs := make([]interface{}, 0, len(orderIDs))
	in := make([]string, 0, len(orderIDs))
	bindNum := 2
	for _, o := range orderIDs {
		IDs = append(IDs, entities.OrderID(o))
		in = append(in, fmt.Sprintf("$%d", bindNum))
		bindNum++
	}
	bind := make([]interface{}, 0, len(in)+1)
	// set all bind vars
	bind = append(bind, marketID)
	bind = append(bind, IDs...)
	// select directly from orders_live table, the current view searches in orders
	// this is used to expire orders, which have to be, by definition, live. This table uses ID as its PK
	// so this is a more optimal way of querying the data.
	query := fmt.Sprintf(`SELECT %s from orders_live WHERE market_id=$1 AND id IN (%s)`, sqlOrderColumns, strings.Join(in, ", "))
	orders := make([]entities.Order, 0, len(orderIDs))
	err := pgxscan.Select(ctx, os.Connection, &orders, query, bind...)

	return orders, os.wrapE(err)
}

func (os *Orders) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByTxHash")()

	orders := []entities.Order{}
	query := fmt.Sprintf(`SELECT %s FROM orders WHERE tx_hash=$1`, sqlOrderColumns)

	err := pgxscan.Select(ctx, os.Connection, &orders, query, txHash)
	if err != nil {
		return nil, fmt.Errorf("querying orders: %w", err)
	}
	return orders, nil
}

// GetByReference returns the last update of orders with the specified user-suppled reference.
func (os *Orders) GetByReferencePaged(ctx context.Context, reference string, p entities.CursorPagination) ([]entities.Order, entities.PageInfo, error) {
	return os.ListOrders(ctx, nil, nil, &reference, false, p, entities.DateRange{}, entities.OrderFilter{})
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
	pagination entities.CursorPagination, alreadyOrdered bool,
) ([]entities.Order, entities.PageInfo, error) {
	var (
		err      error
		orders   []entities.Order
		pageInfo entities.PageInfo
	)
	// This is a bit subtle - if we're selecting from a view that's doing DISTINCT ON ... ORDER BY
	// it is imperative that we don't apply an ORDER BY clause to the outer query or else postgres
	// will try and materialize the entire view; so rely on the view to sort correctly for us.
	ordering := ordersOrdering

	paginateQuery := PaginateQuery[entities.OrderCursor]
	if alreadyOrdered {
		paginateQuery = PaginateQueryWithoutOrderBy[entities.OrderCursor]
	}

	// We don't have views and indexes for iterating backwards for now so we can't use 'last'
	// as it requires us to order in reverse
	if pagination.HasBackward() {
		return nil, entities.PageInfo{}, fmt.Errorf("'last' pagination for orders not currently supported")
	}

	query, args, err = paginateQuery(query, args, ordering, pagination)
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

func currentView(f entities.OrderFilter, p entities.CursorPagination) (string, bool, error) {
	if !p.NewestFirst {
		return "", false, fmt.Errorf("oldest first order query is not currently supported")
	}
	if f.LiveOnly {
		return "orders_live", false, nil
	}
	if f.Reference != nil {
		return "orders_current_desc_by_reference", true, nil
	}
	if len(f.PartyIDs) > 0 {
		return "orders_current_desc_by_party", true, nil
	}
	if len(f.MarketIDs) > 0 {
		return "orders_current_desc_by_market", true, nil
	}
	return "orders_current_desc", true, nil
}

func (os *Orders) ListOrders(
	ctx context.Context,
	p entities.CursorPagination,
	orderFilter entities.OrderFilter,
) ([]entities.Order, entities.PageInfo, error) {
	table, alreadyOrdered, err := currentView(orderFilter, p)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	bind := make([]interface{}, 0, len(orderFilter.PartyIDs)+len(orderFilter.MarketIDs)+1)
	where := strings.Builder{}
	where.WriteString("WHERE 1=1 ")

	whereStr, args := applyOrderFilter(where.String(), bind, orderFilter)

	query := fmt.Sprintf(`SELECT %s from %s %s`, sqlOrderColumns, table, whereStr)
	query, args = filterDateRange(query, ordersFilterDateColumn, ptr.UnBox(orderFilter.DateRange), args...)

	defer metrics.StartSQLQuery("Orders", "GetByMarketPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, args, p, alreadyOrdered)
}

func (os *Orders) ListOrderVersions(ctx context.Context, orderIDStr string, p entities.CursorPagination) ([]entities.Order, entities.PageInfo, error) {
	if orderIDStr == "" {
		return nil, entities.PageInfo{}, errors.New("orderID is required")
	}
	orderID := entities.OrderID(orderIDStr)
	query := fmt.Sprintf(`SELECT %s from orders_current_versions WHERE id=$1`, sqlOrderColumns)
	defer metrics.StartSQLQuery("Orders", "GetByOrderIDPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, []interface{}{orderID}, p, true)
}

func applyOrderFilter(whereClause string, args []any, filter entities.OrderFilter) (string, []any) {
	if filter.ExcludeLiquidity {
		whereClause += " AND COALESCE(lp_id, '') = ''"
	}

	if len(filter.PartyIDs) > 0 {
		parties := strings.Builder{}
		for i, party := range filter.PartyIDs {
			if i > 0 {
				parties.WriteString(",")
			}
			parties.WriteString(nextBindVar(&args, entities.PartyID(party)))
		}
		whereClause += fmt.Sprintf(" AND party_id IN (%s)", parties.String())
	}

	if len(filter.MarketIDs) > 0 {
		markets := strings.Builder{}
		for i, market := range filter.MarketIDs {
			if i > 0 {
				markets.WriteString(",")
			}
			markets.WriteString(nextBindVar(&args, entities.MarketID(market)))
		}
		whereClause += fmt.Sprintf(" AND market_id IN (%s)", markets.String())
	}

	if filter.Reference != nil {
		args = append(args, filter.Reference)
		whereClause += fmt.Sprintf(" AND reference = $%d", len(args))
	}

	if len(filter.Statuses) > 0 {
		states := strings.Builder{}
		for i, status := range filter.Statuses {
			if i > 0 {
				states.WriteString(",")
			}
			states.WriteString(nextBindVar(&args, status))
		}
		whereClause += fmt.Sprintf(" AND status IN (%s)", states.String())
	}

	if len(filter.Types) > 0 {
		types := strings.Builder{}
		for i, orderType := range filter.Types {
			if i > 0 {
				types.WriteString(",")
			}
			types.WriteString(nextBindVar(&args, orderType))
		}
		whereClause += fmt.Sprintf(" AND type IN (%s)", types.String())
	}

	if len(filter.TimeInForces) > 0 {
		timeInForces := strings.Builder{}
		for i, timeInForce := range filter.TimeInForces {
			if i > 0 {
				timeInForces.WriteString(",")
			}
			timeInForces.WriteString(nextBindVar(&args, timeInForce))
		}
		whereClause += fmt.Sprintf(" AND time_in_force IN (%s)", timeInForces.String())
	}

	return whereClause, args
}
