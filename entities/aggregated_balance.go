package entities

import (
	"encoding/hex"
	"strconv"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"github.com/shopspring/decimal"
)

// AggregatedBalance represents the the summed balance of a bunch of accounts at a given
// time. VegaTime and Balance will always be set. The others will be nil unless when
// querying, you requested grouping by one of the corresponding fields.
type AggregatedBalance struct {
	VegaTime  time.Time
	Balance   decimal.Decimal
	AccountID *int64
	PartyID   *[]byte
	AssetID   *[]byte
	MarketID  *[]byte
	Type      *types.AccountType
}

func (balance *AggregatedBalance) ToProto() v2.AggregatedBalance {
	var accountType vega.AccountType
	var accountID, partyID, assetID, marketID *string

	if balance.AccountID != nil {
		aid := strconv.FormatInt(*balance.AccountID, 10)
		accountID = &aid
	}

	if balance.PartyID != nil {
		pid := hex.EncodeToString(*balance.PartyID)
		partyID = &pid
	}

	if balance.AssetID != nil {
		aid := Asset{ID: *balance.AssetID}.HexID()
		assetID = &aid
	}

	if balance.MarketID != nil {
		mid := hex.EncodeToString(*balance.MarketID)
		marketID = &mid
	}

	if balance.Type != nil {
		accountType = *balance.Type
	}

	return v2.AggregatedBalance{
		Timestamp:   balance.VegaTime.UnixNano(),
		Balance:     balance.Balance.String(),
		AccountId:   accountID,
		PartyId:     partyID,
		AssetId:     assetID,
		MarketId:    marketID,
		AccountType: accountType,
	}
}
