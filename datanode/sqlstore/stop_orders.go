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
	"fmt"
	"strings"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
)

type StopOrders struct {
	*ConnectionSource
	batcher MapBatcher[entities.StopOrderKey, entities.StopOrder]
}

var stopOrdersOrdering = TableOrdering{
	ColumnOrdering{Name: "created_at", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: DESC},
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

const (
	stopOrdersFilterDateColumn = "vega_time"
	StopOrdersTableName        = "stop_orders"
)

func NewStopOrders(connectionSource *ConnectionSource) *StopOrders {
	return &StopOrders{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.StopOrderKey, entities.StopOrder](
			StopOrdersTableName, entities.StopOrderColumns),
	}
}

func (so *StopOrders) Add(o entities.StopOrder) error {
	so.batcher.Add(o)
	return nil
}

func (so *StopOrders) Flush(ctx context.Context) ([]entities.StopOrder, error) {
	defer metrics.StartSQLQuery("StopOrders", "Flush")()
	return so.batcher.Flush(ctx, so.Connection)
}

func (so *StopOrders) GetStopOrder(ctx context.Context, orderID string) (entities.StopOrder, error) {
	var err error
	order := entities.StopOrder{}
	id := entities.StopOrderID(orderID)
	defer metrics.StartSQLQuery("StopOrders", "GetStopOrder")()
	query := `select * from stop_orders_current_desc where id=$1`
	err = pgxscan.Get(ctx, so.Connection, &order, query, id)

	return order, so.wrapE(err)
}

func (so *StopOrders) ListStopOrders(ctx context.Context, filter entities.StopOrderFilter, p entities.CursorPagination) ([]entities.StopOrder, entities.PageInfo, error) {
	pageInfo := entities.PageInfo{}
	table, alreadyOrdered, err := stopOrderView(filter, p)
	if err != nil {
		return nil, pageInfo, err
	}

	args := make([]any, 0, len(filter.PartyIDs)+len(filter.MarketIDs)+1)
	where := "WHERE 1=1 "
	whereStr := ""

	whereStr, args = applyStopOrderFilter(where, filter, args...)
	query := fmt.Sprintf("SELECT * FROM %s %s", table, whereStr)
	query, args = filterDateRange(query, stopOrdersFilterDateColumn, ptr.UnBox(filter.DateRange), false, args...)
	defer metrics.StartSQLQuery("StopOrders", "ListStopOrders")()
	return so.queryWithPagination(ctx, query, p, alreadyOrdered, args...)
}

func (so *StopOrders) queryWithPagination(ctx context.Context, query string, p entities.CursorPagination, alreadyOrdered bool, args ...any) ([]entities.StopOrder, entities.PageInfo, error) {
	var (
		err      error
		orders   []entities.StopOrder
		pageInfo entities.PageInfo
	)

	ordering := stopOrdersOrdering
	paginateQuery := PaginateQuery[entities.StopOrderCursor]
	if alreadyOrdered {
		paginateQuery = PaginateQueryWithoutOrderBy[entities.StopOrderCursor]
	}

	// We don't have the necessary views and indexes for iterating backwards for now so we can't use 'last'
	// as it requires us to order in reverse
	if p.HasBackward() {
		return nil, pageInfo, ErrLastPaginationNotSupported
	}

	query, args, err = paginateQuery(query, args, ordering, p)
	if err != nil {
		return orders, pageInfo, err
	}

	err = pgxscan.Select(ctx, so.Connection, &orders, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying stop orders: %w", err)
	}

	orders, pageInfo = entities.PageEntities[*v2.StopOrderEdge](orders, p)
	return orders, pageInfo, nil
}

func stopOrderView(f entities.StopOrderFilter, p entities.CursorPagination) (string, bool, error) {
	if !p.NewestFirst {
		return "", false, fmt.Errorf("oldest first order query is not currently supported")
	}

	if f.LiveOnly {
		return "stop_orders_live", false, nil
	}

	if len(f.PartyIDs) > 0 {
		return "stop_orders_current_desc_by_party", true, nil
	}

	if len(f.MarketIDs) > 0 {
		return "stop_orders_current_desc_by_market", true, nil
	}

	return "stop_orders_current_desc", true, nil
}

func applyStopOrderFilter(where string, filter entities.StopOrderFilter, args ...any) (string, []any) {
	if len(filter.PartyIDs) > 0 {
		parties := strings.Builder{}
		for i, party := range filter.PartyIDs {
			if i > 0 {
				parties.WriteString(",")
			}
			parties.WriteString(nextBindVar(&args, entities.PartyID(party)))
		}
		where += fmt.Sprintf(" AND party_id IN (%s)", parties.String())
	}

	if len(filter.MarketIDs) > 0 {
		markets := strings.Builder{}
		for i, market := range filter.MarketIDs {
			if i > 0 {
				markets.WriteString(",")
			}
			markets.WriteString(nextBindVar(&args, entities.MarketID(market)))
		}
		where += fmt.Sprintf(" AND market_id IN (%s)", markets.String())
	}

	if len(filter.Statuses) > 0 {
		states := strings.Builder{}
		for i, status := range filter.Statuses {
			if i > 0 {
				states.WriteString(",")
			}
			states.WriteString(nextBindVar(&args, status))
		}
		where += fmt.Sprintf(" AND status IN (%s)", states.String())
	}

	if len(filter.ExpiryStrategy) > 0 {
		expiryStrategies := strings.Builder{}
		for i, s := range filter.ExpiryStrategy {
			if i > 0 {
				expiryStrategies.WriteString(",")
			}
			expiryStrategies.WriteString(nextBindVar(&args, s))
		}
		where += fmt.Sprintf(" AND expiry_strategy IN (%s)", expiryStrategies.String())
	}

	return where, args
}
