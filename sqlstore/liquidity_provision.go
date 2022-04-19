package sqlstore

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type LiquidityProvision struct {
	*SQLStore
	batcher MapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision]
}

const (
	sqlOracleLiquidityProvisionColumns = `id, party_id, created_at, updated_at, market_id, 
		commitment_amount, fee, sells, buys, version, status, reference, vega_time`
)

func NewLiquidityProvision(sqlStore *SQLStore) *LiquidityProvision {
	return &LiquidityProvision{
		SQLStore: sqlStore,
		batcher: NewMapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision](
			"liquidity_provisions", entities.LiquidityProvisionColumns),
	}
}

func (lp *LiquidityProvision) Flush(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), lp.conf.Timeout.Duration)
	defer cancel()
	return lp.batcher.Flush(ctx, lp.pool)
}

func (lp *LiquidityProvision) Upsert(liquidityProvision entities.LiquidityProvision) error {
	lp.batcher.Add(liquidityProvision)
	return nil
}

func (lp *LiquidityProvision) Get(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID,
	pagination entities.Pagination) ([]entities.LiquidityProvision, error) {
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

	err := pgxscan.Select(ctx, lp.pool, &liquidityProvisions, query, bindVars...)
	return liquidityProvisions, err
}
