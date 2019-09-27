package collateral

import (
	"code.vegaprotocol.io/vega/internal/events"

	types "code.vegaprotocol.io/vega/proto"
)

type marginUpdate struct {
	events.Transfer
	margin  *types.Account
	general *types.Account
}

// MarginBalance - returns current balance of margin account
func (m marginUpdate) MarginBalance() uint64 {
	return uint64(m.margin.Balance)
}

// GeneralBalance - returns current balance of general account
func (m marginUpdate) GeneralBalance() uint64 {
	return uint64(m.general.Balance)
}

type newOrderMarginUpdate struct {
	events.MarketPosition
	margin   *types.Account
	general  *types.Account
	asset    string
	marketID string
}

func (n newOrderMarginUpdate) Transfer() *types.Transfer {
	return nil
}

func (n newOrderMarginUpdate) Asset() string {
	return n.asset
}

func (n newOrderMarginUpdate) MarketID() string {
	return n.marketID
}

func (n newOrderMarginUpdate) MarginBalance() uint64 {
	return uint64(n.margin.Balance)
}

func (n newOrderMarginUpdate) GeneralBalance() uint64 {
	return uint64(n.general.Balance)
}
