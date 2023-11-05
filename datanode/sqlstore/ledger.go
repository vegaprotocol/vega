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
	"fmt"
	"io"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

var aggregateLedgerEntriesOrdering = TableOrdering{
	ColumnOrdering{Name: "ledger_entry_time", Sorting: ASC},
}

const (
	LedgerMaxDays = 5 * 24 * time.Hour
)

type Ledger struct {
	*ConnectionSource
	batcher ListBatcher[entities.LedgerEntry]
	pending []entities.LedgerEntry
}

func NewLedger(connectionSource *ConnectionSource) *Ledger {
	a := &Ledger{
		ConnectionSource: connectionSource,
		batcher:          NewListBatcher[entities.LedgerEntry]("ledger", entities.LedgerEntryColumns),
	}
	return a
}

func (ls *Ledger) Flush(ctx context.Context) ([]entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "Flush")()

	// This creates an entry time for the ledger entry that is guaranteed to be unique
	// Block event sequence number cannot be used as multiple ledger entries can be created
	// as the result of a single transfer event.
	for i, le := range ls.pending {
		le.LedgerEntryTime = entities.CreateLedgerEntryTime(le.VegaTime, i)
		ls.batcher.Add(le)
	}

	ls.pending = nil

	return ls.batcher.Flush(ctx, ls.Connection)
}

func (ls *Ledger) Add(le entities.LedgerEntry) error {
	ls.pending = append(ls.pending, le)
	return nil
}

func (ls *Ledger) GetByLedgerEntryTime(ctx context.Context, ledgerEntryTime time.Time) (entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "GetByID")()
	le := entities.LedgerEntry{}

	return le, ls.wrapE(pgxscan.Get(ctx, ls.Connection, &le,
		`SELECT ledger_entry_time, quantity, tx_hash, vega_time, transfer_time, type, account_from_balance, account_to_balance
		 FROM ledger WHERE ledger_entry_time =$1`,
		ledgerEntryTime))
}

func (ls *Ledger) GetAll(ctx context.Context) ([]entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "GetAll")()
	ledgerEntries := []entities.LedgerEntry{}
	err := pgxscan.Select(ctx, ls.Connection, &ledgerEntries, `
		SELECT ledger_entry_time, quantity, tx_hash, vega_time, transfer_time, type, account_from_balance, account_to_balance
		FROM ledger`)
	return ledgerEntries, err
}

func (ls *Ledger) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.LedgerEntry, error) {
	ledgerEntries := []entities.LedgerEntry{}
	defer metrics.StartSQLQuery("Ledger", "GetByTxHash")()

	err := pgxscan.Select(ctx, ls.Connection, &ledgerEntries, `
		SELECT ledger_entry_time, account_from_id, account_to_id, quantity, tx_hash, vega_time, transfer_time, type, account_from_balance, account_to_balance
		FROM ledger WHERE tx_hash=$1`,
		txHash,
	)
	return ledgerEntries, err
}

// Return at most 5 days of ledger entries so that we don't have long-running queries that scan through
// too much historic data in timescale to try and match the given date ranges.
func (ls *Ledger) validateDateRange(dateRange entities.DateRange) entities.DateRange {
	if dateRange.Start == nil && dateRange.End == nil {
		return entities.DateRange{
			Start: ptr.From(time.Now().Add(-LedgerMaxDays)),
		}
	}

	if dateRange.Start == nil && dateRange.End != nil {
		return entities.DateRange{
			Start: ptr.From(dateRange.End.Add(-LedgerMaxDays)),
			End:   dateRange.End,
		}
	}

	if (dateRange.Start != nil && dateRange.End == nil) ||
		(dateRange.Start != nil && dateRange.End != nil && dateRange.End.Sub(*dateRange.Start) > LedgerMaxDays) {
		return entities.DateRange{
			Start: dateRange.Start,
			End:   ptr.From(dateRange.Start.Add(LedgerMaxDays)),
		}
	}

	return dateRange
}

// This query requests and sums number of the ledger entries of a given subset of accounts, specified via the 'filter' argument.
// It returns a timeseries (implemented as a list of AggregateLedgerEntry structs), with a row for every time
// the summed ledger entries of the set of specified accounts changes.
// Listed queries should be limited to a single party from each side only. If no or more than one parties are provided
// for sending and receiving accounts - the query returns error.
//
// Entries can be queried by:
//   - listing ledger entries with filtering on the sending account (market_id, asset_id, account_type)
//   - listing ledger entries with filtering on the receiving account (market_id, asset_id, account_type)
//   - listing ledger entries with filtering on the sending AND receiving account
//   - listing ledger entries with filtering on the transfer type (on top of above filters or as a standalone option)
func (ls *Ledger) Query(
	ctx context.Context,
	filter *entities.LedgerEntryFilter,
	dateRange entities.DateRange,
	pagination entities.CursorPagination,
) (*[]entities.AggregatedLedgerEntry, entities.PageInfo, error) {
	var pageInfo entities.PageInfo

	dynamicQuery, whereQuery, args, err := ls.prepareQuery(filter, ls.validateDateRange(dateRange))
	if err != nil {
		return nil, pageInfo, err
	}

	pageQuery, args, err := PaginateQuery[entities.AggregatedLedgerEntriesCursor](
		whereQuery, args, aggregateLedgerEntriesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	query := fmt.Sprintf("%s %s", dynamicQuery, pageQuery)

	defer metrics.StartSQLQuery("Ledger", "Query")()
	rows, err := ls.Connection.Query(ctx, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying ledger entries: %w", err)
	}
	defer rows.Close()

	var results []ledgerEntriesScanned
	if err = pgxscan.ScanAll(&results, rows); err != nil {
		return nil, pageInfo, fmt.Errorf("scanning ledger entries: %w", err)
	}

	ledgerEntries := parseScanned(results)
	res, pageInfo := entities.PageEntities[*v2.AggregatedLedgerEntriesEdge](ledgerEntries, pagination)
	return &res, pageInfo, nil
}

func (ls *Ledger) Export(
	ctx context.Context,
	partyID string,
	assetID *string,
	dateRange entities.DateRange,
	writer io.Writer,
) error {
	if partyID == "" {
		return ErrLedgerEntryExportForParty
	}

	pid := entities.PartyID(partyID)
	pidBytes, err := pid.Bytes()
	if err != nil {
		return fmt.Errorf("invalid party id: %w", err)
	}

	args := []any{pidBytes}
	query := `
		SELECT
			l.vega_time,
			l.quantity,
			CASE
				WHEN ta.party_id = $1 AND fa.party_id != $1 THEN quantity
				WHEN fa.party_id = $1 AND ta.party_id != $1 THEN -quantity
				ELSE 0
			END AS effect,
			l.type AS transfer_type,
			encode(fa.asset_id, 'hex') AS asset_id,
			encode(fa.market_id, 'hex') AS account_from_market_id,
			CASE
				WHEN fa.party_id='\x03' THEN 'network'
				ELSE encode(fa.party_id, 'hex')
				END AS account_from_party_id,
			CASE
				WHEN fa.type=0 THEN 'UNSPECIFIED'
				WHEN fa.type=1 THEN 'INSURANCE'
				WHEN fa.type=2 THEN 'SETTLEMENT'
				WHEN fa.type=3 THEN 'MARGIN'
				WHEN fa.type=4 THEN 'GENERAL'
				WHEN fa.type=5 THEN 'FEES_INFRASTRUCTURE'
				WHEN fa.type=6 THEN 'FEES_LIQUIDITY'
				WHEN fa.type=7 THEN 'FEES_MAKER'
				WHEN fa.type=9 THEN 'BOND'
				WHEN fa.type=10 THEN 'EXTERNAL'
				WHEN fa.type=11 THEN 'GLOBAL_INSURANCE'
				WHEN fa.type=12 THEN 'GLOBAL_REWARD'
				WHEN fa.type=13 THEN 'PENDING_TRANSFERS'
				WHEN fa.type=14 THEN 'REWARD_MAKER_PAID_FEES'
				WHEN fa.type=15 THEN 'REWARD_MAKER_RECEIVED_FEES'
				WHEN fa.type=16 THEN 'REWARD_LP_RECEIVED_FEES'
				WHEN fa.type=17 THEN 'REWARD_MARKET_PROPOSERS'
				WHEN fa.type=18 THEN 'HOLDING'
				WHEN fa.type=19 THEN 'LP_LIQUIDITY_FEES'
				WHEN fa.type=20 THEN 'LIQUIDITY_FEES_BONUS_DISTRIBUTION'
				WHEN fa.type=21 THEN 'NETWORK_TREASURY'
				WHEN fa.type=22 THEN 'VESTING_REWARDS'
				WHEN fa.type=23 THEN 'VESTED_REWARDS'
				WHEN fa.type=24 THEN 'REWARD_AVERAGE_POSITION'
				WHEN fa.type=25 THEN 'REWARD_RELATIVE_RETURN'
				WHEN fa.type=26 THEN 'REWARD_RETURN_VOLATILITY'
				WHEN fa.type=27 THEN 'REWARD_VALIDATOR_RANKING'
				WHEN fa.type=28 THEN 'PENDING_FEE_REFERRAL_REWARD'
				ELSE 'UNKNOWN' END AS account_from_account_type,
			l.account_from_balance AS account_from_balance,
			encode(ta.market_id, 'hex') AS account_to_market_id,
			CASE
				WHEN ta.party_id='\x03' THEN 'network'
				ELSE encode(ta.party_id, 'hex')
				END AS account_to_party_id,
			CASE
				WHEN ta.type=0 THEN 'UNSPECIFIED'
				WHEN ta.type=1 THEN 'INSURANCE'
				WHEN ta.type=2 THEN 'SETTLEMENT'
				WHEN ta.type=3 THEN 'MARGIN'
				WHEN ta.type=4 THEN 'GENERAL'
				WHEN ta.type=5 THEN 'FEES_INFRASTRUCTURE'
				WHEN ta.type=6 THEN 'FEES_LIQUIDITY'
				WHEN ta.type=7 THEN 'FEES_MAKER'
				WHEN ta.type=9 THEN 'BOND'
				WHEN ta.type=10 THEN 'EXTERNAL'
				WHEN ta.type=11 THEN 'GLOBAL_INSURANCE'
				WHEN ta.type=12 THEN 'GLOBAL_REWARD'
				WHEN ta.type=13 THEN 'PENDING_TRANSFERS'
				WHEN ta.type=14 THEN 'REWARD_MAKER_PAID_FEES'
				WHEN ta.type=15 THEN 'REWARD_MAKER_RECEIVED_FEES'
				WHEN ta.type=16 THEN 'REWARD_LP_RECEIVED_FEES'
				WHEN ta.type=17 THEN 'REWARD_MARKET_PROPOSERS'
				WHEN ta.type=18 THEN 'HOLDING'
				WHEN fa.type=19 THEN 'LP_LIQUIDITY_FEES'
				WHEN fa.type=20 THEN 'LIQUIDITY_FEES_BONUS_DISTRIBUTION'
				WHEN fa.type=21 THEN 'NETWORK_TREASURY'
				WHEN fa.type=22 THEN 'VESTING_REWARDS'
				WHEN fa.type=23 THEN 'VESTED_REWARDS'
				WHEN fa.type=24 THEN 'REWARD_AVERAGE_POSITION'
				WHEN fa.type=25 THEN 'REWARD_RELATIVE_RETURN'
				WHEN fa.type=26 THEN 'REWARD_RETURN_VOLATILITY'
				WHEN fa.type=27 THEN 'REWARD_VALIDATOR_RANKING'
				WHEN fa.type=28 THEN 'PENDING_FEE_REFERRAL_REWARD'
				ELSE 'UNKNOWN' END AS account_to_account_type,
			l.account_to_balance AS account_to_balance
		FROM
			ledger l
			INNER JOIN accounts AS fa ON l.account_from_id=fa.id
			INNER JOIN accounts AS ta ON l.account_to_id=ta.id

		WHERE (ta.party_id = $1 OR fa.party_id = $1)
		`

	if assetID != nil {
		id := entities.AssetID(*assetID)
		idBytes, err := id.Bytes()
		if err != nil {
			return fmt.Errorf("invalid asset id: %w", err)
		}
		query = fmt.Sprintf("%s AND fa.asset_id = %s", query, nextBindVar(&args, idBytes))
	}

	if dateRange.Start != nil {
		query = fmt.Sprintf("%s AND l.ledger_entry_time >= %s", query, nextBindVar(&args, dateRange.Start.Format(time.RFC3339)))
	}

	if dateRange.End != nil {
		query = fmt.Sprintf("%s AND l.ledger_entry_time < %s", query, nextBindVar(&args, dateRange.End.Format(time.RFC3339)))
	}

	query = fmt.Sprintf("copy (%s ORDER BY l.ledger_entry_time) to STDOUT (FORMAT csv, HEADER)", query)

	tag, err := ls.Connection.CopyTo(ctx, writer, query, args...)
	if err != nil {
		return fmt.Errorf("copying to stdout: %w", err)
	}

	ls.log.Debug("copy to CSV", logging.Int64("rows affected", tag.RowsAffected()))
	return nil
}

func (*Ledger) prepareQuery(filter *entities.LedgerEntryFilter, dateRange entities.DateRange) (string, string, []any, error) {
	filterQueries, args, err := filterLedgerEntriesQuery(filter)
	if err != nil {
		return "", "", nil, fmt.Errorf("filtering ledger entries: %w", err)
	}

	whereDate := ""
	if dateRange.Start != nil {
		whereDate = fmt.Sprintf("WHERE ledger_entry_time >= %s", nextBindVar(&args, dateRange.Start.Format(time.RFC3339)))
	}

	if dateRange.End != nil {
		if whereDate != "" {
			whereDate = fmt.Sprintf("%s AND", whereDate)
		} else {
			whereDate = "WHERE "
		}
		whereDate = fmt.Sprintf("%s ledger_entry_time < %s", whereDate, nextBindVar(&args, dateRange.End.Format(time.RFC3339)))
	}

	dynamicQuery := createDynamicQuery(filterQueries, filter.CloseOnAccountFilters)

	whereQuery := fmt.Sprintf(`SELECT
			vega_time, quantity, transfer_type, asset_id,
			account_from_market_id, account_from_party_id, account_from_account_type,
			account_to_market_id, account_to_party_id, account_to_account_type,
			account_from_balance, account_to_balance
		FROM entries
		%s`, whereDate)
	return dynamicQuery, whereQuery, args, nil
}

// ledgerEntriesScanned is a local type used as a mediator between pgxscan scanner
// and the AggregatedLedgerEntries types.
// Needed to manually transfer to needed data types that are not accepted by the scanner.
type ledgerEntriesScanned struct {
	VegaTime     time.Time
	Quantity     decimal.Decimal
	TransferType entities.LedgerMovementType
	AssetID      entities.AssetID

	AccountFromPartyID     entities.PartyID
	AccountToPartyID       entities.PartyID
	AccountFromAccountType types.AccountType
	AccountToAccountType   types.AccountType

	AccountFromMarketID entities.MarketID
	AccountToMarketID   entities.MarketID
	AccountFromBalance  decimal.Decimal
	AccountToBalance    decimal.Decimal
}

func parseScanned(scanned []ledgerEntriesScanned) []entities.AggregatedLedgerEntry {
	ledgerEntries := []entities.AggregatedLedgerEntry{}
	if len(scanned) > 0 {
		for i := range scanned {
			ledgerEntries = append(ledgerEntries, entities.AggregatedLedgerEntry{
				VegaTime:            scanned[i].VegaTime,
				Quantity:            scanned[i].Quantity,
				AssetID:             &scanned[i].AssetID,
				FromAccountPartyID:  &scanned[i].AccountFromPartyID,
				ToAccountPartyID:    &scanned[i].AccountToPartyID,
				FromAccountType:     &scanned[i].AccountFromAccountType,
				ToAccountType:       &scanned[i].AccountToAccountType,
				FromAccountMarketID: &scanned[i].AccountFromMarketID,
				ToAccountMarketID:   &scanned[i].AccountToMarketID,
				FromAccountBalance:  scanned[i].AccountFromBalance,
				ToAccountBalance:    scanned[i].AccountToBalance,
			})

			tt := scanned[i].TransferType
			ledgerEntries[i].TransferType = &tt
		}
	}

	return ledgerEntries
}

// createDynamicQuery creates a dynamic query depending on the query cases:
//   - lising all ledger entries without filtering
//   - listing ledger entries with filtering on the sending account
//   - listing ledger entries with filtering on the receiving account
//   - listing ledger entries with filtering on the sending AND receiving account
//   - listing ledger entries with filtering on the transfer type (on top of above filters or as a standalone option)
func createDynamicQuery(filterQueries [3]string, closeOnAccountFilters entities.CloseOnLimitOperation) string {
	whereClause := ""

	tableNameFromAccountQuery := "ledger_entries_from_account_filter"
	query := `
		%s AS (
			SELECT
				ledger.vega_time AS vega_time, ledger.quantity, ledger.type AS transfer_type,
				ledger.account_from_id, ledger.account_to_id,
				ledger.account_from_balance, ledger.account_to_balance,
				account_from.asset_id AS asset_id,
				account_from.party_id AS account_from_party_id,
				account_from.market_id AS account_from_market_id,
				account_from.type AS account_from_account_type,
				account_to.party_id AS account_to_party_id,
				account_to.market_id AS account_to_market_id,
				account_to.type AS account_to_account_type,
				ledger.ledger_entry_time
			FROM ledger
			INNER JOIN accounts AS account_from
			ON ledger.account_from_id=account_from.id
			INNER JOIN accounts AS account_to
			ON ledger.account_to_id=account_to.id),

		entries AS (
			SELECT vega_time, quantity, transfer_type, asset_id,
				account_from_market_id, account_from_party_id, account_from_account_type,
				account_to_market_id, account_to_party_id, account_to_account_type,
				account_from_balance, account_to_balance, ledger_entry_time
			FROM %s
			%s
		)
	`

	tableNameToAccountQuery := "ledger_entries_to_account_filter"
	tableNameCloseOnFilterQuery := "ledger_entries_closed_on_account_filters"
	tableNameOpenOnFilterQuery := "ledger_entries_open_on_account_filters"
	tableNameTransferType := "ledger_entries_transfer_type_filter"

	tableName := ""

	if filterQueries[0] != "" {
		tableName = tableNameFromAccountQuery
		whereClause = fmt.Sprintf("WHERE %s", filterQueries[0])

		if filterQueries[1] != "" {
			if closeOnAccountFilters {
				tableName = tableNameCloseOnFilterQuery
				whereClause = fmt.Sprintf("WHERE (%s) AND (%s)", filterQueries[0], filterQueries[1])
			} else {
				tableName = tableNameOpenOnFilterQuery
				whereClause = fmt.Sprintf("WHERE ((%s) OR (%s))", filterQueries[0], filterQueries[1])
			}
		}
	} else {
		if filterQueries[1] != "" {
			tableName = tableNameToAccountQuery
			whereClause = fmt.Sprintf("WHERE %s", filterQueries[1])
		}
	}

	if filterQueries[2] != "" {
		tableName = tableNameTransferType
		if whereClause != "" {
			whereClause = fmt.Sprintf("%s AND (%s)", whereClause, filterQueries[2])
		} else {
			whereClause = fmt.Sprintf("WHERE %s", filterQueries[2])
		}
	}

	query = fmt.Sprintf(query, tableName, tableName, whereClause)
	query = fmt.Sprintf(`WITH %s`, query)

	return query
}
