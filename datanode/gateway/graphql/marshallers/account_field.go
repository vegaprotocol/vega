// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package marshallers

import (
	"fmt"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/99designs/gqlgen/graphql"
)

func MarshalAccountField(*v2.AccountField) graphql.Marshaler {
	// Nothing returns an account field as of now, but gqlgen wants this method to exist
	panic("Not implemented")
}

func UnmarshalAccountField(i interface{}) (*v2.AccountField, error) {
	v, ok := i.(string)
	if !ok {
		return nil, fmt.Errorf("expected string in account field")
	}

	var af v2.AccountField
	switch {
	case v == "AccountId":
		af = v2.AccountField_ACCOUNT_FIELD_ID
	case v == "PartyId":
		af = v2.AccountField_ACCOUNT_FIELD_PARTY_ID
	case v == "AssetId":
		af = v2.AccountField_ACCOUNT_FIELD_ASSET_ID
	case v == "AccountType":
		af = v2.AccountField_ACCOUNT_FIELD_TYPE
	default:
		return nil, fmt.Errorf("unknown account field %v", v)
	}
	return &af, nil
}
