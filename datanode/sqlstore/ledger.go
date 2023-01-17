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
	"encoding/json"
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

	dynamicQuery := createDynamicQuery(filterQueries, filter.CloseOnAccountFilters)
	queryLedgerEntries := dynamicQuery

	// This pageQuery is the part that gives us the results for the pagination. We will only pass this part of the query
	// to the PaginateQuery function because the WHERE clause in the query above will cause an incorrect SQL statement
	// to be generated
	pageQuery := fmt.Sprintf(`SELECT
			vega_time, quantity, transfer_type, asset_id,
			account_from_market_id, account_from_party_id, account_from_account_type,
			account_to_market_id, account_to_party_id, account_to_account_type,
			account_from_balance, account_to_balance
		FROM entries
		%s`, whereDate)

	pageQuery, args, err = PaginateQuery[entities.AggregatedLedgerEntriesCursor](
		pageQuery, args, aggregateLedgerEntriesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	queryLedgerEntries = fmt.Sprintf("%s %s", queryLedgerEntries, pageQuery)

	defer metrics.StartSQLQuery("Ledger", "Query")()
	rows, err := ls.Connection.Query(ctx, queryLedgerEntries, args...)
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

func (ls *Ledger) Export(
	ctx context.Context,
	partyID, assetID string,
	dateRange entities.DateRange,
	pagination entities.CursorPagination,
) ([]byte, entities.PageInfo, error) {
	if partyID == "" {
		return nil, entities.PageInfo{}, ErrLedgerEntryExportForParty
	}

	filter := &entities.LedgerEntryFilter{
		FromAccountFilter: entities.AccountFilter{
			AssetID:  entities.AssetID(assetID),
			PartyIDs: []entities.PartyID{entities.PartyID(partyID)},
		},
	}

	aggregatedEntries, pageInfo, err := ls.Query(ctx, filter, dateRange, pagination)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	columns := []string{
		"VegaTime", "AssetId", "SenderPartyId", "SenderMarketId", "SenderAccountType",
		"ReceiverPartyId", "ReceiverMarketId", "ReceiverAccountType", "Quantity", "TransferType",
	}

	data, err := json.MarshalIndent(columns, "", " ")
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("failed to prepare columns: %w", err)
	}

	if aggregatedEntries != nil {
		content, err := json.MarshalIndent(*aggregatedEntries, "", "  ")
		if err != nil {
			return nil, entities.PageInfo{}, fmt.Errorf("failed to write content: %w", err)
		}

		data = append(data, content...)
	}

	return data, pageInfo, nil
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
				account_to.type AS account_to_account_type
			FROM ledger
			INNER JOIN accounts AS account_from
			ON ledger.account_from_id=account_from.id
			INNER JOIN accounts AS account_to
			ON ledger.account_to_id=account_to.id),

		entries AS (
			SELECT vega_time, quantity, transfer_type, asset_id,
				account_from_market_id, account_from_party_id, account_from_account_type,
				account_to_market_id, account_to_party_id, account_to_account_type,
				account_from_balance, account_to_balance
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
				whereClause = fmt.Sprintf("WHERE (%s) OR (%s)", filterQueries[0], filterQueries[1])
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
			whereClause = fmt.Sprintf("%s AND %s", whereClause, filterQueries[2])
		} else {
			whereClause = fmt.Sprintf("WHERE %s", filterQueries[2])
		}
	}

	query = fmt.Sprintf(query, tableName, tableName, whereClause)
	query = fmt.Sprintf(`WITH %s`, query)

	return query
}
