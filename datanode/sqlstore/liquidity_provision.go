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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var lpOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
}

type LiquidityProvision struct {
	*ConnectionSource
	batcher  MapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision]
	observer utils.Observer[entities.LiquidityProvision]
}

const (
	sqlOracleLiquidityProvisionColumns = `id, party_id, created_at, updated_at, market_id, 
		commitment_amount, fee, sells, buys, version, status, reference, tx_hash, vega_time`
)

func NewLiquidityProvision(connectionSource *ConnectionSource, log *logging.Logger) *LiquidityProvision {
	return &LiquidityProvision{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision](
			"liquidity_provisions", entities.LiquidityProvisionColumns),
		observer: utils.NewObserver[entities.LiquidityProvision]("liquidity_provisions", log, 10, 10),
	}
}

func (lp *LiquidityProvision) Flush(ctx context.Context) error {
	defer metrics.StartSQLQuery("LiquidityProvision", "Flush")()
	flushed, err := lp.batcher.Flush(ctx, lp.Connection)
	if err != nil {
		return err
	}

	lp.observer.Notify(flushed)
	return nil
}

func (lp *LiquidityProvision) ObserveLiquidityProvisions(ctx context.Context, retries int,
	market *string, party *string,
) (<-chan []entities.LiquidityProvision, uint64) {
	ch, ref := lp.observer.Observe(
		ctx,
		retries,
		func(lp entities.LiquidityProvision) bool {
			marketOk := market == nil || lp.MarketID.String() == *market
			partyOk := party == nil || lp.PartyID.String() == *party
			return marketOk && partyOk
		})
	return ch, ref
}

func (lp *LiquidityProvision) Upsert(ctx context.Context, liquidityProvision entities.LiquidityProvision) error {
	lp.batcher.Add(liquidityProvision)
	return nil
}

func (lp *LiquidityProvision) Get(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	reference string,
	live bool,
	pagination entities.Pagination,
) ([]entities.LiquidityProvision, entities.PageInfo, error) {
	if len(partyID) == 0 && len(marketID) == 0 {
		return nil, entities.PageInfo{}, errors.New("market or party filters are required")
	}

	switch p := pagination.(type) {
	case entities.OffsetPagination:
		return lp.getWithOffsetPagination(ctx, partyID, marketID, reference, live, p)
	case entities.CursorPagination:
		return lp.getWithCursorPagination(ctx, partyID, marketID, reference, live, p)
	default:
		return lp.getWithOffsetPagination(ctx, partyID, marketID, reference, live, entities.OffsetPagination{})
	}
}

func (lp *LiquidityProvision) getWithCursorPagination(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	reference string, live bool, pagination entities.CursorPagination,
) ([]entities.LiquidityProvision, entities.PageInfo, error) {
	query, bindVars := lp.buildLiquidityProvisionsSelect(partyID, marketID, reference, live)

	var err error
	var pageInfo entities.PageInfo
	query, bindVars, err = PaginateQuery[entities.LiquidityProvisionCursor](query, bindVars, lpOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	var liquidityProvisions []entities.LiquidityProvision

	if err = pgxscan.Select(ctx, lp.Connection, &liquidityProvisions, query, bindVars...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	pagedLiquidityProvisions, pageInfo := entities.PageEntities[*v2.LiquidityProvisionsEdge](liquidityProvisions, pagination)
	return pagedLiquidityProvisions, pageInfo, nil
}

func (lp *LiquidityProvision) getWithOffsetPagination(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	reference string, live bool, pagination entities.OffsetPagination) ([]entities.LiquidityProvision,
	entities.PageInfo, error,
) {
	var bindVars []interface{}
	var pageInfo entities.PageInfo

	query, bindVars := lp.buildLiquidityProvisionsSelect(partyID, marketID, reference, live)

	query, bindVars = orderAndPaginateQuery(query, []string{"id", "vega_time"}, pagination, bindVars...)

	var liquidityProvisions []entities.LiquidityProvision

	defer metrics.StartSQLQuery("LiquidityProvision", "Get")()
	err := pgxscan.Select(ctx, lp.Connection, &liquidityProvisions, query, bindVars...)
	return liquidityProvisions, pageInfo, err
}

func (lp *LiquidityProvision) buildLiquidityProvisionsSelect(partyID entities.PartyID, marketID entities.MarketID,
	reference string, live bool,
) (string, []interface{}) {
	var bindVars []interface{}
	selectSQL := ""
	if live {
		selectSQL = fmt.Sprintf(`select %s
from live_liquidity_provisions`, sqlOracleLiquidityProvisionColumns)
	} else {
		selectSQL = fmt.Sprintf(`select %s
from liquidity_provisions`, sqlOracleLiquidityProvisionColumns)
	}

	where := ""

	if partyID != "" {
		where = fmt.Sprintf("%s party_id=%s", where, nextBindVar(&bindVars, partyID))
	}

	if marketID != "" {
		if len(where) > 0 {
			where = where + " and "
		}
		where = fmt.Sprintf("%s market_id=%s", where, nextBindVar(&bindVars, marketID))
	}

	if reference != "" {
		if len(where) > 0 {
			where = where + " and "
		}
		where = fmt.Sprintf("%s reference=%s", where, nextBindVar(&bindVars, reference))
	}

	if len(where) > 0 {
		where = fmt.Sprintf("where %s", where)
	}

	query := fmt.Sprintf(`%s %s`, selectSQL, where)
	return query, bindVars
}
