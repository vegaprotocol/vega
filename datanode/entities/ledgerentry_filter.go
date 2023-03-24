package entities

import (
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

// CloseOnLimitOperation is the type that is used for opening and closing the set of output items under some operation.
// Intended for generic use.
type CloseOnLimitOperation bool

// Settings for receiving closed/open sets on different parts of the outputs of LedgerEntries.
// Any kind of relation between the data types on logical and practical level in the set is the `limit operation`.
// We close or not the set of output items on the limit operation via the `CloseOnOperation` set values.
type LedgerEntryFilter struct {
	// FromAccountFilter is a filter which is used to request properties for FromAccount field.
	FromAccountFilter AccountFilter
	// ToAccountFilter is a filter which is used to request properties for ToAccount field.
	ToAccountFilter AccountFilter

	// Filter on LedgerMovementType
	TransferTypes []LedgerMovementType
	// CloseOnAccountFilters is used to open/close the output set of entries under the FromAccount/ToAccount values.
	// If true -> the output set will contain entries which sending and receiving accounts
	// all match the criteria given in the `AccountFilter` type.
	// Otherwise will contain entries that have a match the settings in both accounts (sending or receiving) or in one of them.
	CloseOnAccountFilters CloseOnLimitOperation
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
	}

	return &filter, nil
}
