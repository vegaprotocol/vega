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

package marshallers

import (
	"errors"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/99designs/gqlgen/graphql"
)

var ErrInvalidLedgerEntryField = errors.New("invalid ledger entry field")

func MarshalLedgerEntryField(*v2.LedgerEntryField) graphql.Marshaler {
	panic("not implemented")
}

func UnmarshalLedgerEntryField(i interface{}) (*v2.LedgerEntryField, error) {
	v, ok := i.(string)
	if !ok {
		return nil, ErrInvalidLedgerEntryField
	}

	var lf v2.LedgerEntryField
	switch {
	case v == "TransferType":
		lf = v2.LedgerEntryField_LEDGER_ENTRY_FIELD_TRANSFER_TYPE
	default:
		return nil, fmt.Errorf("unknown ledger entry field %q", v)
	}
	return &lf, nil
}
