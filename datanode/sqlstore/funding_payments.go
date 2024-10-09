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
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type FundingPayments struct {
	*ConnectionSource
}

var fundingPaymentOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "market_id", Sorting: ASC},
	ColumnOrdering{Name: "funding_period_seq", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
}

func NewFundingPayments(connectionSource *ConnectionSource) *FundingPayments {
	return &FundingPayments{
		ConnectionSource: connectionSource,
	}
}

func (fp *FundingPayments) Add(
	ctx context.Context,
	fundingPayments []*entities.FundingPayment,
) error {
	defer metrics.StartSQLQuery("FundingPayments", "Add")()

	for _, v := range fundingPayments {
		_, err := fp.Exec(ctx,
			`insert into funding_payment(market_id, party_id, funding_period_seq, amount, vega_time, tx_hash, loss_socialisation_amount)
values ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (party_id, market_id, vega_time) DO UPDATE SET
		funding_period_seq=EXCLUDED.funding_period_seq,
		amount=EXCLUDED.amount,
		tx_hash=EXCLUDED.tx_hash,
		loss_socialisation_amount=EXCLUDED.loss_socialisation_amount`,
			v.MarketID, v.PartyID, v.FundingPeriodSeq, v.Amount, v.VegaTime, v.TxHash, v.LossSocialisationAmount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fp *FundingPayments) List(
	ctx context.Context,
	partyID entities.PartyID,
	marketID *entities.MarketID,
	pagination entities.CursorPagination,
) ([]entities.FundingPayment, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("FundingPayments", "List")()
	var fundingPayments []entities.FundingPayment
	var pageInfo entities.PageInfo
	var args []interface{}
	var err error

	query := fmt.Sprintf("select * from funding_payment where party_id = %s", nextBindVar(&args, partyID))

	if marketID != nil {
		query = fmt.Sprintf("%s and market_id = %s", query, nextBindVar(&args, *marketID))
	}

	query, args, err = PaginateQuery[entities.FundingPaymentCursor](query, args, fundingPaymentOrdering, pagination)
	if err != nil {
		return fundingPayments, pageInfo, err
	}

	err = pgxscan.Select(ctx, fp.ConnectionSource, &fundingPayments, query, args...)
	if err != nil {
		return fundingPayments, pageInfo, err
	}

	fundingPayments, pageInfo = entities.PageEntities[*v2.FundingPaymentEdge](fundingPayments, pagination)

	return fundingPayments, pageInfo, nil
}

func (fp *FundingPayments) GetByPartyAndMarket(ctx context.Context, party, market string) (entities.FundingPayment, error) {
	partyID, marketID := entities.PartyID(party), entities.MarketID(market)
	defer metrics.StartSQLQuery("FundingPayments", "GetByPartyAndMarket")()
	var (
		err error
		ret entities.FundingPayment
	)
	query := "SELECT * FROM funding_payment WHERE party_id = $1 AND market_id = $2 ORDER BY vega_time DESC LIMIT 1"
	if err = pgxscan.Select(ctx, fp.ConnectionSource, &ret, query, partyID, marketID); err != nil {
		return ret, err
	}
	return ret, nil
}
