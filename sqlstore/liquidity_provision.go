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
}

const (
	sqlOracleLiquidityProvisionColumns = `id, party_id, created_at, updated_at, market_id, 
		commitment_amount, fee, sells, buys, version, status, reference, vega_time`
)

func NewLiquidityProvision(sqlStore *SQLStore) *LiquidityProvision {
	return &LiquidityProvision{
		SQLStore: sqlStore,
	}
}

func (lp *LiquidityProvision) Upsert(liquidityProvision *entities.LiquidityProvision) error {
	ctx, cancel := context.WithTimeout(context.Background(), lp.conf.Timeout.Duration)
	defer cancel()

	query := fmt.Sprintf(`insert into liquidity_provisions (%s)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
on conflict (id, vega_time) do update
set 
	party_id=EXCLUDED.party_id,
	created_at=EXCLUDED.created_at,
	updated_at=EXCLUDED.updated_at,
	market_id=EXCLUDED.market_id,
	commitment_amount=EXCLUDED.commitment_amount,
	fee=EXCLUDED.fee,
	sells=EXCLUDED.sells,
	buys=EXCLUDED.buys,
	version=EXCLUDED.version,
	status=EXCLUDED.status,
	reference=EXCLUDED.reference`, sqlOracleLiquidityProvisionColumns)

	if _, err := lp.pool.Exec(ctx, query, liquidityProvision.ID, liquidityProvision.PartyID,
		liquidityProvision.CreatedAt, liquidityProvision.UpdatedAt, liquidityProvision.MarketID,
		liquidityProvision.CommitmentAmount, liquidityProvision.Fee, liquidityProvision.Sells,
		liquidityProvision.Buys, liquidityProvision.Version, liquidityProvision.Status,
		liquidityProvision.Reference, liquidityProvision.VegaTime); err != nil {
		return err
	}
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
