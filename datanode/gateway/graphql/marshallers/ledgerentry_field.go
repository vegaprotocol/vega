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
