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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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
	pagination entities.OffsetPagination,
) ([]entities.LiquidityProvision, error) {
	if len(partyID.ID) == 0 && len(marketID.ID) == 0 {
		return nil, errors.New("market or party filters are required")
	}

	var bindVars []interface{}

	selectSql := fmt.Sprintf(`select distinct on (id) %s
from liquidity_provisions`, sqlOracleLiquidityProvisionColumns)

	where := "where"

	if partyID.ID != "" {
		where = fmt.Sprintf("%s party_id=%s", where, nextBindVar(&bindVars, partyID))
	}

	if partyID.ID != "" && marketID.ID != "" {
		where = fmt.Sprintf("%s and", where)
	}

	if marketID.ID != "" {
		where = fmt.Sprintf("%s market_id=%s", where, nextBindVar(&bindVars, marketID))
	}

	query := fmt.Sprintf(`%s %s
order by id, vega_time desc`, selectSql, where)

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)

	var liquidityProvisions []entities.LiquidityProvision

	defer metrics.StartSQLQuery("LiquidityProvision", "Get")()
	err := pgxscan.Select(ctx, lp.Connection, &liquidityProvisions, query, bindVars...)
	return liquidityProvisions, err
}
