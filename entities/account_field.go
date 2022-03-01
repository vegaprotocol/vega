package entities

import (
	"fmt"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
)

// AccountField is an enumeration of the properties of an account
// which can be used for grouping and sorting
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
