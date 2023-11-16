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
	Balance   decimal.Decimal
	AccountID *AccountID
	PartyID   *PartyID
	AssetID   *AssetID
	MarketID  *MarketID
	Type      *types.AccountType
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
