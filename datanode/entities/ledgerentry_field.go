package entities

import (
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

// LedgerEntryField is an enumeration of the properties of a ledger entry
// which can be used for grouping and sorting.
type LedgerEntryField int64

const (
	LedgerEntryFieldUnspecified = iota
	LedgerEntryFieldTransferType
)

func (s LedgerEntryField) String() string {
	switch s {
	case LedgerEntryFieldTransferType:
		return "type"
	}
	return "unknown"
}

func LedgerEntryFieldFromProto(field v2.LedgerEntryField) (LedgerEntryField, error) {
	switch field {
	case v2.LedgerEntryField_LEDGER_ENTRY_FIELD_TRANSFER_TYPE:
		return LedgerEntryFieldTransferType, nil
	default:
		return -1, fmt.Errorf("unknown ledger entry field %q", field)
	}
}
