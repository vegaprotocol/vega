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

// AccountField is an enumeration of the properties of an account
// which can be used for grouping and sorting.
type AccountField int64

const (
	AccountFieldUnspecified = iota
	AccountFieldID
	AccountFieldPartyID
	AccountFieldAssetID
	AccountFieldMarketID
	AccountFieldType
)

func (s AccountField) String() string {
	switch s {
	case AccountFieldID:
		return "account_id"
	case AccountFieldPartyID:
		return "party_id"
	case AccountFieldAssetID:
		return "asset_id"
	case AccountFieldMarketID:
		return "market_id"
	case AccountFieldType:
		return "type"
	}
	return "unknown"
}

func AccountFieldFromProto(field v2.AccountField) (AccountField, error) {
	switch field {
	case v2.AccountField_ACCOUNT_FIELD_ID:
		return AccountFieldID, nil
	case v2.AccountField_ACCOUNT_FIELD_ASSET_ID:
		return AccountFieldAssetID, nil
	case v2.AccountField_ACCOUNT_FIELD_PARTY_ID:
		return AccountFieldPartyID, nil
	case v2.AccountField_ACCOUNT_FIELD_MARKET_ID:
		return AccountFieldMarketID, nil
	case v2.AccountField_ACCOUNT_FIELD_TYPE:
		return AccountFieldType, nil
	default:
		return -1, fmt.Errorf("unknown account field %v", field)
	}
}
