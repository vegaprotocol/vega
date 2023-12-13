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

package entities

import (
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

// CloseOnLimitOperation is the type that is used for opening and closing the set of output items under some operation.
// Intended for generic use.
type CloseOnLimitOperation bool

// LedgerEntryFilter settings for receiving closed/open sets on different parts of the outputs of LedgerEntries.
// Any kind of relation between the data types on logical and practical level in the set is the `limit operation`.
// We close or not the set of output items on the limit operation via the `CloseOnOperation` set values.
type LedgerEntryFilter struct {
	// CloseOnAccountFilters is used to open/close the output set of entries under the FromAccount/ToAccount values.
	// If true -> the output set will contain entries which sending and receiving accounts
	// all match the criteria given in the `AccountFilter` type.
	// Otherwise will contain entries that have a match the settings in both accounts (sending or receiving) or in one of them.
	CloseOnAccountFilters CloseOnLimitOperation
	// FromAccountFilter is a filter which is used to request properties for FromAccount field.
	FromAccountFilter AccountFilter
	// ToAccountFilter is a filter which is used to request properties for ToAccount field.
	ToAccountFilter AccountFilter

	// Filter on LedgerMovementType
	TransferTypes []LedgerMovementType

	// Transfer ID to filter by
	TransferID TransferID
}

func LedgerEntryFilterFromProto(pbFilter *v2.LedgerEntryFilter) (*LedgerEntryFilter, error) {
	filter := LedgerEntryFilter{}
	if pbFilter != nil {
		filter.CloseOnAccountFilters = CloseOnLimitOperation(pbFilter.CloseOnAccountFilters)

		var err error
		filter.FromAccountFilter, err = AccountFilterFromProto(pbFilter.FromAccountFilter)
		if err != nil {
			return nil, err
		}
		filter.ToAccountFilter, err = AccountFilterFromProto(pbFilter.ToAccountFilter)
		if err != nil {
			return nil, err
		}

		if len(pbFilter.TransferTypes) > 0 {
			filter.TransferTypes = make([]LedgerMovementType, len(pbFilter.TransferTypes))
			for i, tt := range pbFilter.TransferTypes {
				t, ok := vega.TransferType_value[tt.String()]
				if ok {
					filter.TransferTypes[i] = LedgerMovementType(t)
				}
			}
		}

		if pbFilter.TransferId != nil {
			filter.TransferID = TransferID(*pbFilter.TransferId)
		}
	}

	return &filter, nil
}
