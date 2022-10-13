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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

var aggregateLedgerEntriesOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

type GroupOptions struct {
	ByAccountField     []entities.AccountField
	ByLedgerEntryField []entities.LedgerEntryField
}

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

func (ls *Ledger) GetByLedgerEntryTime(ledgerEntryTime time.Time) (entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "GetByID")()
	le := entities.LedgerEntry{}
	ctx := context.Background()
	err := pgxscan.Get(ctx, ls.Connection, &le,
		`SELECT ledger_entry_time, quantity, tx_hash, vega_time, transfer_time, type
		 FROM ledger WHERE ledger_entry_time =$1`,
		ledgerEntryTime)
	return le, err
}

func (ls *Ledger) GetAll() ([]entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "GetAll")()
	ctx := context.Background()
	ledgerEntries := []entities.LedgerEntry{}
	err := pgxscan.Select(ctx, ls.Connection, &ledgerEntries, `
		SELECT ledger_entry_time, quantity, tx_hash, vega_time, transfer_time, type
		FROM ledger`)
	return ledgerEntries, err
}

// This query requests and sums number of the ledger entries of a given subset of accounts, specified via the 'filter' argument.
// It returns a timeseries (implemented as a list of AggregateLedgerEntry structs), with a row for every time
// the summed ledger entries of the set of specified accounts changes.
//
// Entries can be queried by:
//   - lising all ledger entries without filtering
//   - listing ledger entries with filtering on the sending account (party_id, market_id, asset_id, account_type)
//   - listing ledger entries with filtering on the receiving account (party_id, market_id, asset_id, account_type)
//   - listing ledger entries with filtering on the sending AND receiving account
//   - listing ledger entries with filtering on the transfer type (on top of above filters or as a standalone option)
func (ls *Ledger) Query(
	filter *entities.LedgerEntryFilter,
	groupOptions *GroupOptions,
	dateRange entities.DateRange,
	pagination entities.CursorPagination,
) (*[]entities.AggregatedLedgerEntries, entities.PageInfo, error) {
	var pageInfo entities.PageInfo

	filterQueries, args, err := filterLedgerEntriesQuery(filter)
	if err != nil {
		return nil, pageInfo, err
	}

	whereDate := ""
	if dateRange.Start != nil {
		whereDate = fmt.Sprintf("WHERE vega_time >= %s", nextBindVar(&args, *dateRange.Start))
	}

	if dateRange.End != nil {
		if whereDate != "" {
			whereDate = fmt.Sprintf("%s AND", whereDate)
		} else {
			whereDate = "WHERE "
		}
		whereDate = fmt.Sprintf("%s vega_time < %s", whereDate, nextBindVar(&args, *dateRange.End))
	}

	var (
		groupBy      string
		groupColumns []string
	)

	if groupOptions != nil {
		groupColumns = prepareGroupFields(groupOptions.ByAccountField, groupOptions.ByLedgerEntryField)

		if len(groupColumns) > 0 {
			for i, col := range groupColumns {
				if i == 0 {
					groupBy = fmt.Sprintf(`%s,`, col)
				} else {
					groupBy = fmt.Sprintf(`%s%s,`, groupBy, col)
				}
			}
		}
	}

	dynamicQuery := createDynamicQuery(filterQueries, filter.CloseOnAccountFilters, groupBy)
	queryLedgerEntries := dynamicQuery

	// This pageQuery is the part that gives us the results for the pagination. We will only pass this part of the query
	// to the PaginateQuery function because the WHERE clause in the query above will cause an incorrect SQL statement
	// to be generated
	pageQuery := fmt.Sprintf(`SELECT vega_time, %s quantity
		FROM entries
		%s`, groupBy, whereDate)

	pageQuery, args, err = PaginateQuery[entities.AggregatedLedgerEntriesCursor](
		pageQuery, args, aggregateLedgerEntriesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	queryLedgerEntries = fmt.Sprintf("%s %s", queryLedgerEntries, pageQuery)

	defer metrics.StartSQLQuery("Ledger", "Query")()
	rows, err := ls.Connection.Query(context.Background(), queryLedgerEntries, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying ledger entries: %w", err)
	}
	defer rows.Close()

	results := []ledgerEntriesScanned{}
	err = pgxscan.ScanAll(&results, rows)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("scanning ledger entries: %w", err)
	}

	ledgerEntries := parseScanned(results)
	res, pageInfo := entities.PageEntities[*v2.AggregatedLedgerEntriesEdge](ledgerEntries, pagination)
	return &res, pageInfo, nil
}

// ledgerEntriesScanned is a local type used as a mediator between pgxscan scanner
// and the AggregatedLedgerEntries types.
// Needed to manually transfer to needed data types that are not accepted by the scanner.
type ledgerEntriesScanned struct {
	ID       int64
	VegaTime time.Time

	Quantity     decimal.Decimal
	AccountType  *types.AccountType
	TransferType entities.LedgerMovementType

	PartyID  *entities.PartyID
	AssetID  *entities.AssetID
	MarketID *entities.MarketID
}

func parseScanned(scanned []ledgerEntriesScanned) []entities.AggregatedLedgerEntries {
	ledgerEntries := []entities.AggregatedLedgerEntries{}
	if len(scanned) > 0 {
		for i, s := range scanned {
			ledgerEntries = append(ledgerEntries, entities.AggregatedLedgerEntries{
				VegaTime: s.VegaTime,
				Quantity: s.Quantity,
				PartyID:  s.PartyID,
				AssetID:  s.AssetID,
				MarketID: s.MarketID,
			})

			if s.AccountType != nil {
				ledgerEntries[i].AccountType = s.AccountType
			}

			tt := s.TransferType
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
func createDynamicQuery(filterQueries [3]string, closeOnAccountFilters entities.CloseOnLimitOperation, groupBy string) string {
	tableNameGeneralQuery := "ledger_entries"
	generalQuery := `
		%s AS (
			SELECT ledger.vega_time, ledger.account_from_id, ledger.account_to_id, ledger.quantity, ledger.type AS transfer_type,
				accounts.id, accounts.asset_id, accounts.market_id, accounts.party_id, accounts.type AS account_type
			FROM ledger
			INNER JOIN accounts
			ON ledger.account_from_id=accounts.id),
	`

	tableNameAccountFromQuery := "ledger_entries_account_from_filter"
	accountFromQuery := `
		%s AS (
			SELECT ledger.vega_time, ledger.account_from_id, ledger.account_to_id, ledger.quantity, ledger.type AS transfer_type,
				accounts.id, accounts.asset_id, accounts.market_id, accounts.party_id, accounts.type AS account_type
			FROM ledger
			INNER JOIN accounts
			ON ledger.account_from_id=accounts.id WHERE %s),
	`

	tableNameAccountToQuery := "ledger_entries_account_to_filter"
	accountToQuery := `
		%s AS (
			SELECT ledger.vega_time, ledger.account_from_id, ledger.account_to_id, ledger.quantity, ledger.type AS transfer_type,
				accounts.id, accounts.asset_id, accounts.market_id, accounts.party_id, accounts.type AS account_type
			FROM ledger
			INNER JOIN accounts
			ON ledger.account_to_id=accounts.id WHERE %s),
	`

	tableNameCloseOnFilterQuery := "ledger_entries_closed_on_account_filters"
	closeOnAccountFilterQuery := `
		%s AS (
			SELECT DISTINCT
			ledger_entries_account_from_filter.vega_time,
			ledger_entries_account_from_filter.account_from_id,
			ledger_entries_account_from_filter.account_to_id,
			ledger_entries_account_from_filter.quantity,
			ledger_entries_account_from_filter.transfer_type,
			ledger_entries_account_from_filter.asset_id,
			ledger_entries_account_from_filter.market_id,
			ledger_entries_account_from_filter.party_id,
			ledger_entries_account_from_filter.account_type,
			ledger_entries_account_from_filter.account_to_id
			FROM  %s
			INNER JOIN %s
			ON %s.account_to_id=%s.account_to_id
			),
`

	query := ""
	tableName := ""

	groupQuery := `
		entries AS (
			SELECT SUM(quantity) AS quantity, %s vega_time
				FROM %s
				WHERE quantity IS NOT NULL
				GROUP BY %s vega_time
				ORDER BY %s vega_time
			)`

	if filterQueries[0] != "" {
		query = fmt.Sprintf(accountFromQuery, tableNameAccountFromQuery, filterQueries[0])
		tableName = tableNameAccountFromQuery

		if filterQueries[1] != "" {
			accountToQuery = fmt.Sprintf(accountToQuery, tableNameAccountToQuery, filterQueries[1])
			query = fmt.Sprintf(`%s %s`, query, accountToQuery)

			if closeOnAccountFilters {
				closeOnAccountFilterQuery = fmt.Sprintf(
					closeOnAccountFilterQuery,
					tableNameCloseOnFilterQuery,
					tableNameAccountFromQuery,
					tableNameAccountToQuery,
					tableNameAccountFromQuery,
					tableNameAccountToQuery,
				)

				query = fmt.Sprintf(`%s %s`, query, closeOnAccountFilterQuery)
				tableName = tableNameCloseOnFilterQuery
			} else {
				tableNameUnionAccountQuery := "ledger_entries_union_filter"
				unionFiltersQuery := `
		%s AS (
			SELECT DISTINCT ON (vega_time)
					vega_time, account_from_id, account_to_id, quantity, transfer_type, asset_id, market_id, party_id, account_type
				FROM (
				SELECT DISTINCT ON (vega_time)
					vega_time, account_from_id, account_to_id, quantity, transfer_type, asset_id, market_id, party_id, account_type
				FROM  %s

				UNION
				SELECT DISTINCT ON (vega_time)
					vega_time, account_from_id, account_to_id, quantity, transfer_type, asset_id, market_id, party_id, account_type
				FROM %s
			) AS u
		),
`

				unionFiltersQuery = fmt.Sprintf(
					unionFiltersQuery,
					tableNameUnionAccountQuery,
					tableNameAccountFromQuery,
					tableNameAccountToQuery,
				)

				query = fmt.Sprintf(`%s %s`, query, unionFiltersQuery)
				tableName = tableNameUnionAccountQuery
			}
		}
	} else {
		if filterQueries[1] != "" {
			query = fmt.Sprintf(accountToQuery, tableNameAccountToQuery, filterQueries[1])
			tableName = tableNameAccountToQuery
		}
	}

	// general case: table_name is still "entries"
	if query == "" {
		query = fmt.Sprintf(generalQuery, tableNameGeneralQuery)
		tableName = tableNameGeneralQuery
	}

	if filterQueries[2] != "" {
		// Add transferType filtering
		tableNameTransferType := "ledger_entries_transfer_type_filter"
		transferTypeQuery := `
		%s AS (
			SELECT
				vega_time, quantity, transfer_type, asset_id, market_id, party_id, account_type
			FROM %s
			WHERE %s),
`

		transferTypeQuery = fmt.Sprintf(
			transferTypeQuery,
			tableNameTransferType,
			tableName,
			filterQueries[2],
		)
		tableName = tableNameTransferType

		query = fmt.Sprintf(`%s %s`, query, transferTypeQuery)
	}

	groupQuery = fmt.Sprintf(groupQuery, groupBy, tableName, groupBy, groupBy)
	query = fmt.Sprintf(`WITH %s %s`, query, groupQuery)

	return query
}
