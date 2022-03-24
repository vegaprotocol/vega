package entities

import (
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	*Account
	Balance decimal.Decimal
}

func (ab *AccountBalance) ToProto() *vega.Account {
	return &vega.Account{
		Owner:    ab.PartyID.String(),
		Balance:  ab.Balance.String(),
		Asset:    ab.AssetID.String(),
		MarketId: ab.MarketID.String(),
		Type:     ab.Account.Type,
	}
}
