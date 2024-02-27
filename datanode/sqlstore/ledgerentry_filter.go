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
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"

	"golang.org/x/exp/maps"
)

var (
	ErrLedgerEntryFilterForParty = errors.New("filtering ledger entries should be limited to a single party")
	ErrLedgerEntryExportForParty = errors.New("exporting ledger entries should be limited to a single party")
)

// Return an SQL query string and corresponding bind arguments to return
// ledger entries rows resulting from different filter options.
func filterLedgerEntriesQuery(filter *entities.LedgerEntryFilter, args *[]interface{}, whereClauses *[]string) error {
	if err := handlePartiesFiltering(filter); err != nil {
		return err
	}

	fromAccountDBQuery, err := accountFilterToDBQuery(filter.FromAccountFilter, args, "account_from.")
	if err != nil {
		return fmt.Errorf("invalid fromAccount filters: %w", err)
	}

	toAccountDBQuery, err := accountFilterToDBQuery(filter.ToAccountFilter, args, "account_to.")
	if err != nil {
		return fmt.Errorf("invalid toAccount filters: %w", err)
	}

	accountTransferTypeDBQuery := transferTypeFilterToDBQuery(filter.TransferTypes)

	if fromAccountDBQuery != "" {
		if toAccountDBQuery != "" {
			if filter.CloseOnAccountFilters {
				*whereClauses = append(*whereClauses, fromAccountDBQuery, toAccountDBQuery)
			} else {
				*whereClauses = append(*whereClauses, fmt.Sprintf("((%s) OR (%s))", fromAccountDBQuery, toAccountDBQuery))
			}
		} else {
			*whereClauses = append(*whereClauses, fromAccountDBQuery)
		}
	} else if toAccountDBQuery != "" {
		*whereClauses = append(*whereClauses, toAccountDBQuery)
	}

	if accountTransferTypeDBQuery != "" {
		*whereClauses = append(*whereClauses, accountTransferTypeDBQuery)
	}

	return nil
}

// accountFilterToDBQuery creates a DB query section string from the given account filter values.
func accountFilterToDBQuery(af entities.AccountFilter, args *[]interface{}, prefix string) (string, error) {
	var err error

	whereClauses := []string{}

	// Asset filtering
	if af.AssetID.String() != "" {
		assetIDAsBytes, err := af.AssetID.Bytes()
		if err != nil {
			return "", fmt.Errorf("invalid asset id: %w", err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("account_from.asset_id=%s", nextBindVar(args, assetIDAsBytes)))
	}

	// Party filtering
	if len(af.PartyIDs) == 1 {
		partyIDAsBytes, err := af.PartyIDs[0].Bytes()
		if err != nil {
			return "", fmt.Errorf("invalid party id: %w", err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf(`%sparty_id=%s`, prefix, nextBindVar(args, partyIDAsBytes)))
	}

	// Market filtering
	if len(af.MarketIDs) > 0 {
		marketIds := make([][]byte, len(af.MarketIDs))
		for i, market := range af.MarketIDs {
			marketIds[i], err = market.Bytes()
			if err != nil {
				return "", fmt.Errorf("invalid market id: %w", err)
			}
		}

		whereClauses = append(whereClauses, fmt.Sprintf("%smarket_id=ANY(%s)", prefix, nextBindVar(args, marketIds)))
	}

	// Account types filtering
	if len(af.AccountTypes) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf(`%stype=ANY(%s)`, prefix, nextBindVar(args, getUniqueAccountTypes(af.AccountTypes))))
	}

	return strings.Join(whereClauses, " AND "), nil
}

func getUniqueAccountTypes(accountTypes []vega.AccountType) []vega.AccountType {
	accountTypesList := []vega.AccountType{}
	accountTypesMap := map[vega.AccountType]struct{}{}
	for _, at := range accountTypes {
		_, ok := accountTypesMap[at]
		if ok {
			continue
		}
		accountTypesMap[at] = struct{}{}
		accountTypesList = append(accountTypesList, at)
	}

	return accountTypesList
}

func transferTypeFilterToDBQuery(transferTypeFilter []entities.LedgerMovementType) string {
	if len(transferTypeFilter) == 0 {
		return ""
	}

	transferTypesMap := map[entities.LedgerMovementType]string{}
	for _, transferType := range transferTypeFilter {
		if _, alreadyRegistered := transferTypesMap[transferType]; alreadyRegistered {
			continue
		}
		value, valid := vega.TransferType_name[int32(transferType)]
		if !valid {
			continue
		}

		transferTypesMap[transferType] = "'" + value + "'"
	}

	if len(transferTypesMap) == 0 {
		return ""
	}

	return "ledger.type IN (" + strings.Join(maps.Values(transferTypesMap), ", ") + ")"
}

func handlePartiesFiltering(filter *entities.LedgerEntryFilter) error {
	var partyIDFrom entities.PartyID
	var partyIDTo entities.PartyID

	if len(filter.FromAccountFilter.PartyIDs) > 1 || len(filter.ToAccountFilter.PartyIDs) > 1 {
		return ErrLedgerEntryFilterForParty
	}

	if len(filter.FromAccountFilter.PartyIDs) > 0 {
		partyIDFrom = filter.FromAccountFilter.PartyIDs[0]
	}

	if len(filter.ToAccountFilter.PartyIDs) > 0 {
		partyIDTo = filter.ToAccountFilter.PartyIDs[0]
	}

	if partyIDFrom == "" && partyIDTo == "" && filter.TransferID == "" {
		return ErrLedgerEntryFilterForParty
	}

	return nil
}
