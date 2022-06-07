package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

const (
	sqlOrderColumns = `id, market_id, party_id, side, price,
                       size, remaining, time_in_force, type, status,
                       reference, reason, version, batch_id, pegged_offset,
                       pegged_reference, lp_id, created_at, updated_at, expires_at,
                       vega_time, seq_num`
)

type Orders struct {
	*ConnectionSource
	batcher MapBatcher[entities.OrderKey, entities.Order]
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

// GetAll returns all updates to all orders (including changes to orders that don't increment the version number)
func (os *Orders) GetAll(ctx context.Context) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetAll")()
	orders := []entities.Order{}
	query := fmt.Sprintf("SELECT %s FROM orders", sqlOrderColumns)
	err := pgxscan.Select(ctx, os.Connection, &orders, query)
	return orders, err
}

// GetByOrderId returns the last update of the order with the given ID
func (os *Orders) GetByOrderID(ctx context.Context, orderIdStr string, version *int32) (entities.Order, error) {
	var err error
	order := entities.Order{}
	orderId := entities.NewOrderID(orderIdStr)

	defer metrics.StartSQLQuery("Orders", "GetByOrderID")()
	if version != nil && *version > 0 {
		query := fmt.Sprintf("SELECT %s FROM orders_current_versions WHERE id=$1 and version=$2", sqlOrderColumns)
		err = pgxscan.Get(ctx, os.Connection, &order, query, orderId, version)
	} else {
		query := fmt.Sprintf("SELECT %s FROM orders_current WHERE id=$1", sqlOrderColumns)
		err = pgxscan.Get(ctx, os.Connection, &order, query, orderId)
	}
	return order, err
}

// GetByMarket returns the last update of the all the orders in a particular market
func (os *Orders) GetByMarket(ctx context.Context, marketIdStr string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByMarket")()
	marketId := entities.NewMarketID(marketIdStr)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE market_id=$1`, sqlOrderColumns)
	args := []interface{}{marketId}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByParty returns the last update of the all the orders in a particular party
func (os *Orders) GetByParty(ctx context.Context, partyIdStr string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByParty")()
	partyId := entities.NewPartyID(partyIdStr)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE party_id=$1`, sqlOrderColumns)
	args := []interface{}{partyId}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByReference returns the last update of orders with the specified user-suppled reference
func (os *Orders) GetByReference(ctx context.Context, reference string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetByReference")()
	query := fmt.Sprintf(`SELECT %s from orders_current WHERE reference=$1`, sqlOrderColumns)
	args := []interface{}{reference}
	return os.queryOrders(ctx, query, args, &p)
}

// GetAllVersionsByOrderID the last update to all versions (e.g. manual changes that lead to
// incrementing the version field) of a given order id.
func (os *Orders) GetAllVersionsByOrderID(ctx context.Context, id string, p entities.OffsetPagination) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetAllVersionsByOrderID")()
	query := fmt.Sprintf(`SELECT %s from orders_current_versions WHERE id=$1`, sqlOrderColumns)
	args := []interface{}{entities.NewOrderID(id)}
	return os.queryOrders(ctx, query, args, &p)
}

// GetLiveOrders fetches all currently live orders so the market depth data can be rebuilt
// from the orders data in the database
func (os *Orders) GetLiveOrders(ctx context.Context) ([]entities.Order, error) {
	defer metrics.StartSQLQuery("Orders", "GetLiveOrders")()
	query := fmt.Sprintf(`select %s from orders_current
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
	pagination entities.Pagination,
) ([]entities.Order, entities.PageInfo, error) {
	var (
		err      error
		vegaTime time.Time
		seqNum   uint64
	)

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	var builders CursorQueryParameters

	if cursor != "" {
		vegaTime, seqNum, err = entities.ParseOrderCursor(cursor)
		if err != nil {
			return nil, entities.PageInfo{}, fmt.Errorf("parsing cursor: %w", err)
		}
	}

	builders = []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, vegaTime),
		NewCursorQueryParameter("seq_num", sorting, cmp, seqNum),
	}

	query, args = orderAndPaginateWithCursor(query, pagination, builders, args...)
	var orders []entities.Order
	var pageInfo entities.PageInfo
	var pagedOrders []entities.Order
	err = pgxscan.Select(ctx, os.Connection, &orders, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying orders: %w", err)
	}

	pagedOrders, pageInfo = entities.PageEntities(orders, pagination)
	return pagedOrders, pageInfo, nil
}

func paginateOrderQuery(query string, args []interface{}, p entities.OffsetPagination) (string, []interface{}) {
	dir := "ASC"
	if p.Descending {
		dir = "DESC"
	}

	var limit interface{} = nil
	if p.Limit != 0 {
		limit = p.Limit
	}

	query = fmt.Sprintf(" %s ORDER BY vega_time %s, id %s LIMIT %s OFFSET %s",
		query, dir, dir, nextBindVar(&args, limit), nextBindVar(&args, p.Skip))

	return query, args
}

func (os *Orders) GetByMarketPaged(ctx context.Context, marketIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	if marketIDStr == "" {
		return nil, entities.PageInfo{}, errors.New("marketID is required")
	}

	marketID := entities.NewMarketID(marketIDStr)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE market_id=$1`, sqlOrderColumns)
	defer metrics.StartSQLQuery("Orders", "GetByMarketPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, []interface{}{marketID}, p)
}

func (os *Orders) GetByPartyPaged(ctx context.Context, partyIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	if partyIDStr == "" {
		return nil, entities.PageInfo{}, errors.New("partyID is required")
	}

	partyID := entities.NewPartyID(partyIDStr)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE party_id=$1`, sqlOrderColumns)
	defer metrics.StartSQLQuery("Orders", "GetByPartyPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, []interface{}{partyID}, p)
}

func (os *Orders) GetOrderVersionsByIDPaged(ctx context.Context, orderIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	if orderIDStr == "" {
		return nil, entities.PageInfo{}, errors.New("orderID is required")
	}
	orderID := entities.NewOrderID(orderIDStr)
	query := fmt.Sprintf(`SELECT %s from orders_current_versions WHERE id=$1`, sqlOrderColumns)
	defer metrics.StartSQLQuery("Orders", "GetByOrderIDPaged")()

	return os.queryOrdersWithCursorPagination(ctx, query, []interface{}{orderID}, p)
}

func (os *Orders) GetByPartyAndMarketPaged(ctx context.Context, partyIDStr, marketIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	if partyIDStr == "" {
		return nil, entities.PageInfo{}, errors.New("partyID is required")
	}

	partyID := entities.NewPartyID(partyIDStr)

	args := make([]interface{}, 0)
	args = append(args, partyID)

	query := fmt.Sprintf(`SELECT %s from orders_current WHERE party_id=$1`, sqlOrderColumns)

	if marketIDStr != "" {
		marketID := entities.NewMarketID(marketIDStr)
		args = append(args, marketID)
		query = fmt.Sprintf("%s AND market_id=$2", query)
	}

	defer metrics.StartSQLQuery("Orders", "GetByPartyAndMarketID")()

	return os.queryOrdersWithCursorPagination(ctx, query, args, p)
}
