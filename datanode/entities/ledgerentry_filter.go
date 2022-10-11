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
	// CloseOnAccountFilters is used to open/close the output set of entries under the AccountFrom/AccountTo values.
	// If true -> the output set will contain entries which sending and receiving accounts
	// all match the criteria given in the `AccountFilter` type.
	// Otherwise will contain entries that have a match the settings in both accounts (sending or receiving) or in one of them.
	CloseOnAccountFilters CloseOnLimitOperation
	// AccountFromFilters is a list of filters which is used to request properties for AccountFrom field.
	AccountFromFilters []AccountFilter
	// AccountToFilters is a list of filters which is used to request properties for AccountTo field.
	AccountToFilters []AccountFilter

	// Filter on LedgerMovementType
	TransferTypes []LedgerMovementType
}

func LedgerEntryFilterFromProto(pbFilter *v2.LedgerEntryFilter) (*LedgerEntryFilter, error) {
	filter := LedgerEntryFilter{}
	if pbFilter != nil {
		filter.CloseOnAccountFilters = CloseOnLimitOperation(pbFilter.CloseOnAccountFilters)

		if len(pbFilter.AccountFromFilters) > 0 {
			filter.AccountFromFilters = make([]AccountFilter, len(pbFilter.AccountFromFilters))
			for i, afp := range pbFilter.AccountFromFilters {
				af, err := AccountFilterFromProto(afp)
				if err != nil {
					return nil, err
				}

				filter.AccountFromFilters[i] = af
			}
		}

		if len(pbFilter.AccountToFilters) > 0 {
			filter.AccountToFilters = make([]AccountFilter, len(pbFilter.AccountToFilters))
			for i, afp := range pbFilter.AccountToFilters {
				af, err := AccountFilterFromProto(afp)
				if err != nil {
					return nil, err
				}

				filter.AccountToFilters[i] = af
			}
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
