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
