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
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

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
	case v == "MarketId":
		af = v2.AccountField_ACCOUNT_FIELD_MARKET_ID
	case v == "AssetId":
		af = v2.AccountField_ACCOUNT_FIELD_ASSET_ID
	case v == "AccountType":
		af = v2.AccountField_ACCOUNT_FIELD_TYPE
	default:
		return nil, fmt.Errorf("unknown account field %v", v)
	}
	return &af, nil
}
