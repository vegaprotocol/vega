package collateral

import (
	"code.vegaprotocol.io/vega/events"

	types "code.vegaprotocol.io/vega/proto"
)

type marginUpdate struct {
	events.MarketPosition
	margin   *types.Account
	general  *types.Account
	asset    string
	marketID string
}

func (n marginUpdate) Transfer() *types.Transfer {
	return nil
}

func (n marginUpdate) Asset() string {
	return n.asset
}

func (n marginUpdate) MarketID() string {
	return n.marketID
}

func (n marginUpdate) MarginBalance() uint64 {
	if n.margin == nil {
		return 0
	}
	return uint64(n.margin.Balance)
}

func (n marginUpdate) GeneralBalance() uint64 {
	if n.general == nil {
		return 0
	}
	return uint64(n.general.Balance)
}
