package sqlstore

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
)

var ErrLedgerEntryFilterForParty = errors.New("filtering ledger entries should be limited to a single party")

// Return an SQL query string and corresponding bind arguments to return
// ledger entries rows resulting from different filter options.
func filterLedgerEntriesQuery(filter *entities.LedgerEntryFilter) ([3]string, []interface{}, error) {
	err := handlePartiesFiltering(filter)
	if err != nil {
		return [3]string{}, nil, err
	}

	var args []interface{}
	filterQueries := [3]string{}

	// AccountFrom filter
	accountFromDBQuery, nargs, err := accountFilterToDBQuery(filter.AccountFromFilter, &args)
	if err != nil {
		return [3]string{}, nil, fmt.Errorf("error parsing accountFrom filter values: %w", err)
	}
	args = *nargs

	// AccountTo filter
	accountToDBQuery, nargs, err := accountFilterToDBQuery(filter.AccountToFilter, &args)
	if err != nil {
		return [3]string{}, nil, fmt.Errorf("error parsing accountFrom filter values: %w", err)
	}
	args = *nargs

	// TransferTypeFilters
	accountTransferTypeDBQuery := transferTypeFilterToDBQuery(filter.TransferTypes, &args)

	filterQueries[0] = accountFromDBQuery
	filterQueries[1] = accountToDBQuery
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
	if len(af.PartyIDs) == 1 {
		partyIDAsBytes, err := af.PartyIDs[0].Bytes()
		if err != nil {
			return "", nil, fmt.Errorf("invalid party id: %w", err)
		}
		if singleAccountFilter != "" {
			singleAccountFilter = fmt.Sprintf(`%s AND party_id=%s`, singleAccountFilter, nextBindVar(args, partyIDAsBytes))
		} else {
			singleAccountFilter = fmt.Sprintf(`party_id=%s`, nextBindVar(args, partyIDAsBytes))
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

func handlePartiesFiltering(filter *entities.LedgerEntryFilter) error {
	var partyIDFrom entities.PartyID
	var partyIDTo entities.PartyID

	if len(filter.AccountFromFilter.PartyIDs) > 1 || len(filter.AccountToFilter.PartyIDs) > 1 {
		return ErrLedgerEntryFilterForParty
	}

	if len(filter.AccountFromFilter.PartyIDs) > 0 {
		partyIDFrom = filter.AccountFromFilter.PartyIDs[0]
	}

	if len(filter.AccountToFilter.PartyIDs) > 0 {
		partyIDTo = filter.AccountToFilter.PartyIDs[0]
	}

	if partyIDFrom == "" && partyIDTo == "" {
		return ErrLedgerEntryFilterForParty
	}

	return nil
}
