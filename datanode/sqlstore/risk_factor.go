// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/georgysavva/scany/pgxscan"
)

type RiskFactors struct {
	*ConnectionSource
}

const (
	sqlRiskFactorColumns = `market_id, short, long, tx_hash, vega_time`
)

func NewRiskFactors(connectionSource *ConnectionSource) *RiskFactors {
	return &RiskFactors{
		ConnectionSource: connectionSource,
	}
}

func (rf *RiskFactors) Upsert(ctx context.Context, factor *entities.RiskFactor) error {
	defer metrics.StartSQLQuery("RiskFactor", "Upsert")()
	query := fmt.Sprintf(`insert into risk_factors (%s)
values ($1, $2, $3, $4, $5)
on conflict (market_id, vega_time) do update
set
	short=EXCLUDED.short,
	long=EXCLUDED.long,
	tx_hash=EXCLUDED.tx_hash`, sqlRiskFactorColumns)

	if _, err := rf.Connection.Exec(ctx, query, factor.MarketID, factor.Short, factor.Long, factor.TxHash, factor.VegaTime); err != nil {
		err = fmt.Errorf("could not insert risk factor into database: %w", err)
		return err
	}

	return nil
}

func (rf *RiskFactors) GetMarketRiskFactors(ctx context.Context, marketID string) (entities.RiskFactor, error) {
	defer metrics.StartSQLQuery("RiskFactors", "GetMarketRiskFactors")()
	var riskFactor entities.RiskFactor
	var bindVars []interface{}

	query := fmt.Sprintf(`select %s
		from risk_factors_current
		where market_id = %s`, sqlRiskFactorColumns, nextBindVar(&bindVars, entities.MarketID(marketID)))

	return riskFactor, rf.wrapE(pgxscan.Get(ctx, rf.Connection, &riskFactor, query, bindVars...))
}
