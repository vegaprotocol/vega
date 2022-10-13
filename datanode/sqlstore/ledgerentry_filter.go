package sqlstore

import (
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
)

// Return an SQL query string and corresponding bind arguments to return
// ledger entries rows resulting from different filter options.
func filterLedgerEntriesQuery(filter *entities.LedgerEntryFilter) ([3]string, []interface{}, error) {
	var args []interface{}

	filterQueries := [3]string{}

	// AccountFrom filter
	accountFromFilters := []string{}
	if len(filter.AccountFromFilters) > 0 {
		for _, af := range filter.AccountFromFilters {
			singleAccountDBQuery, nargs, err := accountFilterToDBQuery(af, &args)
			args = *nargs

			if err != nil {
				return [3]string{}, nil, fmt.Errorf("error parsing accountFrom filter values: %w", err)
			}

			if singleAccountDBQuery != "" {
				accountFromFilters = append(accountFromFilters, singleAccountDBQuery)
			}
		}
	}

	// AccountTo filter
	accountToFilters := []string{}
	if len(filter.AccountToFilters) > 0 {
		for _, af := range filter.AccountToFilters {
			singleAccountDBQuery, nargs, err := accountFilterToDBQuery(af, &args)
			args = *nargs

			if err != nil {
				return [3]string{}, nil, fmt.Errorf("error parsing accountTo filter values: %w", err)
			}

			if singleAccountDBQuery != "" {
				accountToFilters = append(accountToFilters, singleAccountDBQuery)
			}
		}
	}

	// Example:
	// (asset_id=$7 AND party_id=ANY($8) AND market_id=ANY($9)) OR (asset_id=$10 AND market_id=ANY($11)) OR (asset_id=$12)
	accountsFromDBQuery := ""
	if len(accountFromFilters) > 0 {
		for i, af := range accountFromFilters {
			accountsFromDBQuery = fmt.Sprintf(`%s (%s)`, accountsFromDBQuery, af)
			if i < len(accountFromFilters)-1 {
				accountsFromDBQuery = fmt.Sprintf(`%s OR`, accountsFromDBQuery)
			}
		}
	}

	// Example:
	// (asset_id=$7 AND party_id=ANY($8) AND market_id=ANY($9)) OR (asset_id=$10 AND market_id=ANY($11)) OR (asset_id=$12)
	accountsToDBQuery := ""
	if len(accountToFilters) > 0 {
		for i, af := range accountToFilters {
			accountsToDBQuery = fmt.Sprintf(`%s (%s)`, accountsToDBQuery, af)
			if i < len(accountToFilters)-1 {
				accountsToDBQuery = fmt.Sprintf(`%s OR`, accountsToDBQuery)
			}
		}
	}

	// TransferTypeFilters
	accountTransferTypeDBQuery := transferTypeFilterToDBQuery(filter.TransferTypes, &args)

	filterQueries[0] = accountsFromDBQuery
	filterQueries[1] = accountsToDBQuery
	filterQueries[2] = accountTransferTypeDBQuery

	return filterQueries, args, nil
}

// accountFilterToDBQuery creates a DB query section string from the given account filter values.
func accountFilterToDBQuery(af entities.AccountFilter, args *[]interface{}) (string, *[]interface{}, error) {
	var (
		singleAccountFilter string
		err                 error
	)

	// Asset filtering
	if af.AssetID.String() != "" {
		singleAccountFilter = fmt.Sprintf("%sasset_id=%s", singleAccountFilter, nextBindVar(args, af.AssetID))
	}

	// Party filtering
	if len(af.PartyIDs) > 0 {
		partyIDs := make([][]byte, len(af.PartyIDs))
		for i, party := range af.PartyIDs {
			partyIDs[i], err = party.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("invalid party id: %w", err)
			}
		}
		if singleAccountFilter != "" {
			singleAccountFilter = fmt.Sprintf(`%s AND party_id=ANY(%s)`, singleAccountFilter, nextBindVar(args, partyIDs))
		} else {
			singleAccountFilter = fmt.Sprintf(`party_id=ANY(%s)`, nextBindVar(args, partyIDs))
		}
	}

	// Market filtering
	if len(af.MarketIDs) > 0 {
		marketIds := make([][]byte, len(af.MarketIDs))
		for i, market := range af.MarketIDs {
			marketIds[i], err = market.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("invalid market id: %w", err)
			}
		}

		if singleAccountFilter != "" {
			singleAccountFilter = fmt.Sprintf(`%s AND market_id=ANY(%s)`, singleAccountFilter, nextBindVar(args, marketIds))
		} else {
			singleAccountFilter = fmt.Sprintf(`market_id=ANY(%s)`, nextBindVar(args, marketIds))
		}
	}

	// Account types filtering
	if len(af.AccountTypes) > 0 {
		if singleAccountFilter != "" {
			singleAccountFilter = fmt.Sprintf(`%s AND accounts.type=ANY(%s)`, singleAccountFilter, nextBindVar(args, af.AccountTypes))
		} else {
			singleAccountFilter = fmt.Sprintf(`accounts.type=ANY(%s)`, nextBindVar(args, af.AccountTypes))
		}
	}

	return singleAccountFilter, args, nil
}

func transferTypeFilterToDBQuery(transferTypeFilter []entities.LedgerMovementType, args *[]interface{}) string {
	transferTypeFilterString := ""
	if len(transferTypeFilter) > 0 {
		for i, transferType := range transferTypeFilter {
			_, ok := vega.TransferType_name[int32(transferType)]
			if !ok {
				continue
			}

			transferTypeFilterString = fmt.Sprintf(`%stransfer_type=%s`, transferTypeFilterString, nextBindVar(args, transferType))
			if i < len(transferTypeFilter)-1 {
				transferTypeFilterString = fmt.Sprintf(`%s OR `, transferTypeFilterString)
			}
		}
	}

	return transferTypeFilterString
}

// prepareGroupFields checks columns provided for grouping results.
func prepareGroupFields(groupByAccountField []entities.AccountField, groupByLedgerEntryField []entities.LedgerEntryField) []string {
	fields := []string{}

	for _, col := range groupByAccountField {
		if col.String() == "type" {
			fields = append(fields, "account_type")
		} else {
			fields = append(fields, col.String())
		}
	}

	for _, col := range groupByLedgerEntryField {
		if col.String() == "type" {
			fields = append(fields, "transfer_type")
		} else {
			fields = append(fields, col.String())
		}
	}

	return fields
}
