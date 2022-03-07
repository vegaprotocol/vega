package entities

import (
	"encoding/hex"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	*Account
	Balance decimal.Decimal
}

func (ab *AccountBalance) ToProto() *vega.Account {
	owner := ""
	market := ""

	if len(ab.PartyID) > 0 {
		owner = hex.EncodeToString(ab.PartyID)
	}

	asset := Asset{
		ID: ab.AssetID,
	}

	if len(ab.MarketID) > 0 {
		market = hex.EncodeToString(ab.MarketID)
	}

	return &vega.Account{
		Owner:    owner,
		Balance:  ab.Balance.String(),
		Asset:    asset.HexID(),
		MarketId: market,
		Type:     ab.Account.Type,
	}
}
