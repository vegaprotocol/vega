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

package sqlstore

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type LiquidityProvision struct {
	*ConnectionSource
	batcher MapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision]
}

const (
	sqlOracleLiquidityProvisionColumns = `id, party_id, created_at, updated_at, market_id, 
		commitment_amount, fee, sells, buys, version, status, reference, vega_time`
)

func NewLiquidityProvision(connectionSource *ConnectionSource) *LiquidityProvision {
	return &LiquidityProvision{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision](
			"liquidity_provisions", entities.LiquidityProvisionColumns),
	}
}

func (lp *LiquidityProvision) Flush(ctx context.Context) error {
	defer metrics.StartSQLQuery("LiquidityProvision", "Flush")()
	_, err := lp.batcher.Flush(ctx, lp.pool)
	return err
}

func (lp *LiquidityProvision) Upsert(ctx context.Context, liquidityProvision entities.LiquidityProvision) error {
	lp.batcher.Add(liquidityProvision)
	return nil
}

func (lp *LiquidityProvision) Get(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	reference string,
	pagination entities.Pagination,
) ([]entities.LiquidityProvision, entities.PageInfo, error) {
	if len(partyID.ID) == 0 && len(marketID.ID) == 0 {
		return nil, entities.PageInfo{}, errors.New("market or party filters are required")
	}

	switch p := pagination.(type) {
	case entities.OffsetPagination:
		return lp.getWithOffsetPagination(ctx, partyID, marketID, reference, p)
	case entities.CursorPagination:
		return lp.getWithCursorPagination(ctx, partyID, marketID, reference, p)
	default:
		return lp.getWithOffsetPagination(ctx, partyID, marketID, reference, entities.OffsetPagination{})
	}

}

func (lp *LiquidityProvision) getWithCursorPagination(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	reference string, pagination entities.CursorPagination) ([]entities.LiquidityProvision, entities.PageInfo, error) {

	query, bindVars := lp.buildLiquidityProvisionsSelect(partyID, marketID, reference)

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	var err error
	lc := &entities.LiquidityProvisionCursor{}

	if cursor != "" {
		err = lc.Parse(cursor)
		if err != nil {
			return nil, entities.PageInfo{}, fmt.Errorf("parsing cursor: %w", err)
		}
	}

	builders := []CursorQueryParameter{
		NewCursorQueryParameter("id", sorting, cmp, entities.NewLiquidityProvisionID(lc.ID)),
		NewCursorQueryParameter("vega_time", sorting, cmp, lc.VegaTime),
	}

	query, bindVars = orderAndPaginateWithCursor(query, pagination, builders, bindVars...)

	var liquidityProvisions []entities.LiquidityProvision

	if err := pgxscan.Select(ctx, lp.Connection, &liquidityProvisions, query, bindVars...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	pagedLiquidityProvisions, pageInfo := entities.PageEntities[*v2.LiquidityProvisionsEdge](liquidityProvisions, pagination)
	return pagedLiquidityProvisions, pageInfo, nil

}

func (lp *LiquidityProvision) getWithOffsetPagination(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	reference string, pagination entities.OffsetPagination) ([]entities.LiquidityProvision,
	entities.PageInfo, error) {
	var bindVars []interface{}
	var pageInfo entities.PageInfo

	query, bindVars := lp.buildLiquidityProvisionsSelect(partyID, marketID, reference)

	query, bindVars = orderAndPaginateQuery(query, []string{"id", "vega_time"}, pagination, bindVars...)

	var liquidityProvisions []entities.LiquidityProvision

	defer metrics.StartSQLQuery("LiquidityProvision", "Get")()
	err := pgxscan.Select(ctx, lp.Connection, &liquidityProvisions, query, bindVars...)
	return liquidityProvisions, pageInfo, err
}

func (lp *LiquidityProvision) buildLiquidityProvisionsSelect(partyID entities.PartyID, marketID entities.MarketID,
	reference string) (string, []interface{}) {
	var bindVars []interface{}

	selectSql := fmt.Sprintf(`select %s
from current_liquidity_provisions`, sqlOracleLiquidityProvisionColumns)

	where := ""

	if partyID.ID != "" {
		where = fmt.Sprintf("%s party_id=%s", where, nextBindVar(&bindVars, partyID))
	}

	if marketID.ID != "" {
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

	query := fmt.Sprintf(`%s %s`, selectSql, where)
	return query, bindVars
}
