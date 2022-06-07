package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type AccountBalance struct {
	*Account
	Balance  decimal.Decimal
	VegaTime time.Time
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

type AccountBalanceKey struct {
	AccountID int64
	VegaTime  time.Time
}

func (b AccountBalance) Key() AccountBalanceKey {
	return AccountBalanceKey{b.Account.ID, b.VegaTime}
}

func (b AccountBalance) ToRow() []interface{} {
	return []interface{}{b.Account.ID, b.VegaTime, b.Balance}
}
