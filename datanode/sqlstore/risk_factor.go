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
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type RiskFactors struct {
	*ConnectionSource
}

const (
	sqlRiskFactorColumns = `market_id, short, long, vega_time`
)

func NewRiskFactors(connectionSource *ConnectionSource) *RiskFactors {
	return &RiskFactors{
		ConnectionSource: connectionSource,
	}
}

func (rf *RiskFactors) Upsert(ctx context.Context, factor *entities.RiskFactor) error {
	defer metrics.StartSQLQuery("RiskFactor", "Upsert")()
	query := fmt.Sprintf(`insert into risk_factors (%s)
values ($1, $2, $3, $4)
on conflict (market_id, vega_time) do update
set 
	short=EXCLUDED.short,
	long=EXCLUDED.long`, sqlRiskFactorColumns)

	if _, err := rf.Connection.Exec(ctx, query, factor.MarketID, factor.Short, factor.Long, factor.VegaTime); err != nil {
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
		from risk_factors
		where market_id = %s`, sqlRiskFactorColumns, nextBindVar(&bindVars, entities.NewMarketID(marketID)))

	err := pgxscan.Get(ctx, rf.Connection, &riskFactor, query, bindVars...)

	return riskFactor, err
}
