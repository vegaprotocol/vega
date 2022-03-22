package sqlstore

import (
	"context"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type RiskFactors struct {
	*SQLStore
}

const (
	sqlRiskFactorColumns = `market_id, short, long, vega_time`
)

func NewRiskFactors(sqlStore *SQLStore) *RiskFactors {
	return &RiskFactors{
		SQLStore: sqlStore,
	}
}

func (rf *RiskFactors) Upsert(factor *entities.RiskFactor) error {
	ctx, cancel := context.WithTimeout(context.Background(), rf.conf.Timeout.Duration)
	defer cancel()

	query := fmt.Sprintf(`insert into risk_factors (%s)
values ($1, $2, $3, $4)
on conflict (market_id, vega_time) do update
set 
	short=EXCLUDED.short,
	long=EXCLUDED.long`, sqlRiskFactorColumns)

	if _, err := rf.pool.Exec(ctx, query, factor.MarketID, factor.Short, factor.Long, factor.VegaTime); err != nil {
		err = fmt.Errorf("could not insert risk factor into database: %w", err)
		return err
	}

	return nil
}

func (rf *RiskFactors) GetMarketRiskFactors(ctx context.Context, marketID string) (entities.RiskFactor, error) {
	market, err := hex.DecodeString(marketID)
	if err != nil {
		return entities.RiskFactor{}, fmt.Errorf("bad market ID (must be a hex string): %w", err)
	}

	var riskFactor entities.RiskFactor
	var bindVars []interface{}

	query := fmt.Sprintf(`select %s
		from risk_factors
		where market_id = %s`, sqlRiskFactorColumns, nextBindVar(&bindVars, market))

	err = pgxscan.Get(ctx, rf.pool, &riskFactor, query, bindVars...)

	return riskFactor, err
}
