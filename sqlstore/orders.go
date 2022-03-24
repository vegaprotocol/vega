package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Orders struct {
	*SQLStore
}

func NewOrders(sqlStore *SQLStore) *Orders {
	a := &Orders{
		SQLStore: sqlStore,
	}
	return a
}

// Add inserts an order update row into the database if an row for this (block time, order id, version)
// does not already exist; otherwise update the existing row with information supplied.
// Currently we only store the last update to an order per block, so the order history is not
// complete if multiple updates happen in one block.
func (ps *Orders) Add(ctx context.Context, o entities.Order) error {
	_, err := ps.pool.Exec(ctx,
		`INSERT INTO orders(
			id, market_id, party_id, side, price,
			size, remaining, time_in_force, type, status,
			reference, reason, version, pegged_offset, batch_id,
			pegged_reference, lp_id, created_at, updated_at, expires_at, vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9, $10,
				 $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		 ON CONFLICT (id, version, vega_time) DO UPDATE SET
			market_id=EXCLUDED.market_id,
			party_id=EXCLUDED.party_id,
			side=EXCLUDED.side,
			price=EXCLUDED.price,
			size=EXCLUDED.size,
			remaining=EXCLUDED.remaining,
			time_in_force=EXCLUDED.time_in_force,
			type=EXCLUDED.type,
			status=EXCLUDED.status,
			reference=EXCLUDED.reference,
			reason=EXCLUDED.reason,
			version=EXCLUDED.version,
			pegged_offset=EXCLUDED.pegged_offset,
			batch_id=EXCLUDED.batch_id,
			pegged_reference=EXCLUDED.pegged_reference,
			lp_id=EXCLUDED.lp_id,
			created_at=EXCLUDED.created_at,
			updated_at=EXCLUDED.updated_at,
			expires_at=EXCLUDED.expires_at;`,
		o.ID, o.MarketID, o.PartyID, o.Side, o.Price,
		o.Size, o.Remaining, o.TimeInForce, o.Type, o.Status,
		o.Reference, o.Reason, o.Version, o.PeggedOffset, o.BatchID,
		o.PeggedReference, o.LpID, o.CreatedAt, o.UpdatedAt, o.ExpiresAt, o.VegaTime)
	return err
}

// GetAll returns all updates to all orders (including changes to orders that don't increment the version number)
func (os *Orders) GetAll(ctx context.Context) ([]entities.Order, error) {
	orders := []entities.Order{}
	err := pgxscan.Select(ctx, os.pool, &orders, `
		SELECT * from orders;`)
	return orders, err
}

// GetByOrderId returns the last update of the order with the given ID
func (os *Orders) GetByOrderID(ctx context.Context, orderIdStr string, version *int32) (entities.Order, error) {
	var err error
	order := entities.Order{}
	orderId := entities.NewOrderID(orderIdStr)

	if version != nil && *version > 0 {
		err = pgxscan.Get(ctx, os.pool, &order, `SELECT * FROM orders_current_versions WHERE id=$1 and version=$2`, orderId, version)
	} else {
		err = pgxscan.Get(ctx, os.pool, &order, `SELECT * FROM orders_current WHERE id=$1`, orderId)
	}
	return order, err
}

// GetByMarket returns the last update of the all the orders in a particular market
func (os *Orders) GetByMarket(ctx context.Context, marketIdStr string, p entities.Pagination) ([]entities.Order, error) {
	marketId := entities.NewMarketID(marketIdStr)

	query := `SELECT * from orders_current WHERE market_id=$1`
	args := []interface{}{marketId}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByParty returns the last update of the all the orders in a particular party
func (os *Orders) GetByParty(ctx context.Context, partyIdStr string, p entities.Pagination) ([]entities.Order, error) {
	partyId := entities.NewPartyID(partyIdStr)

	query := `SELECT * from orders_current WHERE party_id=$1`
	args := []interface{}{partyId}
	return os.queryOrders(ctx, query, args, &p)
}

// GetByReference returns the last update of orders with the specified user-suppled reference
func (os *Orders) GetByReference(ctx context.Context, reference string, p entities.Pagination) ([]entities.Order, error) {
	query := `SELECT * from orders_current WHERE reference=$1`
	args := []interface{}{reference}
	return os.queryOrders(ctx, query, args, &p)
}

// GetAllVersionsByOrderID the last update to all versions (e.g. manual changes that lead to
// incrementing the version field) of a given order id.
func (os *Orders) GetAllVersionsByOrderID(ctx context.Context, id string, p entities.Pagination) ([]entities.Order, error) {
	query := `SELECT * from orders_current_versions WHERE id=$1`
	args := []interface{}{entities.NewOrderID(id)}
	return os.queryOrders(ctx, query, args, &p)
}

// -------------------------------------------- Utility Methods

func (os *Orders) queryOrders(ctx context.Context, query string, args []interface{}, p *entities.Pagination) ([]entities.Order, error) {
	if p != nil {
		query, args = paginateOrderQuery(query, args, *p)
	}

	orders := []entities.Order{}
	err := pgxscan.Select(ctx, os.pool, &orders, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying orders: %w", err)
	}
	return orders, nil
}

func paginateOrderQuery(query string, args []interface{}, p entities.Pagination) (string, []interface{}) {
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
