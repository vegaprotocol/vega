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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/shopspring/decimal"
)

// AggregatedBalance represents the the summed balance of a bunch of accounts at a given
// time. VegaTime and Balance will always be set. The others will be nil unless when
// querying, you requested grouping by one of the corresponding fields.
type AggregatedBalance struct {
	VegaTime  time.Time
	AccountID *AccountID
	PartyID   *PartyID
	AssetID   *AssetID
	MarketID  *MarketID
	Type      *types.AccountType
	Balance   decimal.Decimal
}

// NewAggregatedBalanceFromValues returns a new AggregatedBalance from a list of values as returned
// from pgx.rows.values().
// - vegaTime is assumed to be first
// - then any extra fields listed in 'fields' in order (usually as as result of grouping)
// - then finally the balance itself.
func AggregatedBalanceScan(fields []AccountField, rows interface {
	Next() bool
	Values() ([]any, error)
},
) ([]AggregatedBalance, error) {
	// Iterate through the result set
	balances := []AggregatedBalance{}
	for rows.Next() {
		var ok bool
		bal := AggregatedBalance{}
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		bal.VegaTime, ok = values[0].(time.Time)
		if !ok {
			return nil, fmt.Errorf("unable to cast to time.Time: %v", values[0])
		}

		for i, field := range fields {
			if field == AccountFieldType {
				intAccountType, ok := values[i+1].(int32)
				if !ok {
					return nil, fmt.Errorf("unable to cast to integer account type: %v", values[i])
				}
				accountType := types.AccountType(intAccountType)
				bal.Type = &accountType
				continue
			}

			idBytes, ok := values[i+1].([]byte)
			if !ok {
				return nil, fmt.Errorf("unable to cast to []byte: %v", values[i])
			}

			switch field {
			case AccountFieldID:
				var id AccountID
				id.SetBytes(idBytes)
				bal.AccountID = &id
			case AccountFieldPartyID:
				var id PartyID
				id.SetBytes(idBytes)
				bal.PartyID = &id
			case AccountFieldAssetID:
				var id AssetID
				id.SetBytes(idBytes)
				bal.AssetID = &id
			case AccountFieldMarketID:
				var id MarketID
				id.SetBytes(idBytes)
				bal.MarketID = &id
			default:
				return nil, fmt.Errorf("invalid field: %v", field)
			}
		}
		lastValue := values[len(values)-1]

		if bal.Balance, ok = lastValue.(decimal.Decimal); !ok {
			return nil, fmt.Errorf("unable to cast to decimal %v", lastValue)
		}

		balances = append(balances, bal)
	}

	return balances, nil
}

func (balance *AggregatedBalance) ToProto() *v2.AggregatedBalance {
	var partyID, assetID, marketID *string

	if balance.PartyID != nil {
		pid := balance.PartyID.String()
		partyID = &pid
	}

	if balance.AssetID != nil {
		aid := balance.AssetID.String()
		assetID = &aid
	}

	if balance.MarketID != nil {
		mid := balance.MarketID.String()
		marketID = &mid
	}

	return &v2.AggregatedBalance{
		Timestamp:   balance.VegaTime.UnixNano(),
		Balance:     balance.Balance.String(),
		PartyId:     partyID,
		AssetId:     assetID,
		MarketId:    marketID,
		AccountType: balance.Type,
	}
}

func (balance AggregatedBalance) Cursor() *Cursor {
	return NewCursor(AggregatedBalanceCursor{
		VegaTime:  balance.VegaTime,
		AccountID: balance.AccountID,
		PartyID:   balance.PartyID,
		AssetID:   balance.AssetID,
		MarketID:  balance.MarketID,
		Type:      balance.Type,
	}.String())
}

func (balance AggregatedBalance) ToProtoEdge(_ ...any) (*v2.AggregatedBalanceEdge, error) {
	return &v2.AggregatedBalanceEdge{
		Node:   balance.ToProto(),
		Cursor: balance.Cursor().Encode(),
	}, nil
}

type AggregatedBalanceCursor struct {
	VegaTime  time.Time `json:"vega_time"`
	AccountID *AccountID
	PartyID   *PartyID
	AssetID   *AssetID
	MarketID  *MarketID
	Type      *types.AccountType
}

func (c AggregatedBalanceCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal aggregate balance cursor: %w", err))
	}
	return string(bs)
}

func (c *AggregatedBalanceCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
