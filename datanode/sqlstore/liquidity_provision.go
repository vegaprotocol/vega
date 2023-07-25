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

	"code.vegaprotocol.io/vega/protos/vega"

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

var providerOrdering = TableOrdering{
	ColumnOrdering{Name: "market_id", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
	ColumnOrdering{Name: "ordinality", Sorting: ASC},
}

type LiquidityProvision struct {
	*ConnectionSource
	batcher  MapBatcher[entities.LiquidityProvisionKey, entities.LiquidityProvision]
	observer utils.Observer[entities.LiquidityProvision]
}

type LiquidityProviderFeeShare struct {
	Ordinality            int64
	MarketID              entities.MarketID
	PartyID               string
	AverageLiquidityScore string `db:"average_score"`
	EquityLikeShare       string
	AverageEntryValuation string
	VirtualStake          string
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
	case entities.CursorPagination:
		return lp.getWithCursorPagination(ctx, partyID, marketID, reference, live, p)
	default:
		panic("unsupported pagination")
	}
}

func (lp *LiquidityProvision) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.LiquidityProvision, error) {
	defer metrics.StartSQLQuery("LiquidityProvision", "GetByTxHash")()

	var liquidityProvisions []entities.LiquidityProvision
	query := fmt.Sprintf(`SELECT %s FROM liquidity_provisions WHERE tx_hash = $1`, sqlOracleLiquidityProvisionColumns)

	err := pgxscan.Select(ctx, lp.Connection, &liquidityProvisions, query, txHash)
	if err != nil {
		return nil, err
	}

	return liquidityProvisions, nil
}

func (lp *LiquidityProvision) ListProviders(ctx context.Context, partyID *entities.PartyID,
	marketID *entities.MarketID, pagination entities.CursorPagination) (
	[]entities.LiquidityProvider, entities.PageInfo, error,
) {
	var pageInfo entities.PageInfo
	var feeShares []LiquidityProviderFeeShare
	var err error

	if partyID == nil && marketID == nil {
		return nil, pageInfo, errors.New("market, party or both filters are required")
	}

	query, args := buildLiquidityProviderFeeShareQuery(partyID, marketID)

	query, args, err = PaginateQuery[entities.LiquidityProviderCursor](query, args, providerOrdering, pagination)

	if err != nil {
		return nil, pageInfo, err
	}

	providers := []entities.LiquidityProvider{}

	err = pgxscan.Select(ctx, lp.Connection, &feeShares, query, args...)
	if err != nil {
		return nil, pageInfo, err
	}

	for _, feeShare := range feeShares {
		providers = append(providers, entities.LiquidityProvider{
			Ordinality: feeShare.Ordinality,
			PartyID:    entities.PartyID(feeShare.PartyID),
			MarketID:   feeShare.MarketID,
			FeeShare: &vega.LiquidityProviderFeeShare{
				Party:                 feeShare.PartyID,
				EquityLikeShare:       feeShare.EquityLikeShare,
				AverageEntryValuation: feeShare.AverageEntryValuation,
				AverageScore:          feeShare.AverageLiquidityScore,
				VirtualStake:          feeShare.VirtualStake,
			},
		})
	}

	providers, pageInfo = entities.PageEntities[*v2.LiquidityProviderEdge](providers, pagination)

	return providers, pageInfo, nil
}

func buildLiquidityProviderFeeShareQuery(partyID *entities.PartyID, marketID *entities.MarketID) (string, []interface{}) {
	args := []interface{}{}

	// The lp data is available in the current market data table
	subQuery := `
select
    ordinality,
	cmd.market,
	coalesce(lpfs.fee_share ->> 'party', '')                   as party,
	coalesce(lpfs.fee_share ->> 'average_score', '')           as average_score,
	coalesce(lpfs.fee_share ->> 'equity_like_share', '')       as equity_like_share,
	coalesce(lpfs.fee_share ->> 'average_entry_valuation', '') as average_entry_valuation,
	coalesce(lpfs.fee_share ->> 'virtual_stake', '') 		   as virtual_stake
from current_market_data cmd,
jsonb_array_elements(liquidity_provider_fee_shares) with ordinality lpfs(fee_share, ordinality)
where liquidity_provider_fee_shares != 'null' and liquidity_provider_fee_shares is not null
`

	if partyID != nil {
		subQuery = fmt.Sprintf("%s and decode(lpfs.fee_share ->>'party', 'hex') = %s", subQuery, nextBindVar(&args, partyID))
	}

	// if a specific market is requested, then filter by that market too
	if marketID != nil {
		subQuery = fmt.Sprintf("%s and cmd.market = %s", subQuery, nextBindVar(&args, *marketID))
	}

	// we join with the live liquidity providers table to make sure we are only returning data
	// for liquidity providers that are currently active
	query := fmt.Sprintf(`WITH liquidity_provider_fee_share(ordinality, market_id, party_id, average_score, equity_like_share, average_entry_valuation, virtual_stake) as (%s)
        SELECT fs.ordinality, fs.market_id, fs.party_id, fs.average_score, fs.equity_like_share, fs.average_entry_valuation, fs.virtual_stake
	    FROM liquidity_provider_fee_share fs
        JOIN live_liquidity_provisions lps ON encode(lps.party_id, 'hex') = fs.party_id
        	AND lps.market_id = fs.market_id`, subQuery)

	return query, args
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
