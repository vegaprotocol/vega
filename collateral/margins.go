package collateral

import (
	"code.vegaprotocol.io/vega/events"

	types "code.vegaprotocol.io/vega/proto"
)

type marginUpdate struct {
	events.MarketPosition
	margin   *types.Account
	general  *types.Account
	lock     *types.Account
	bond     *types.Account
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

// GeneralBalance here we cumulate both the general
// account and bon account so other package do not have
// to worry about how much funds are available in both
// if a bond account exists
// TODO(): maybe rename this method into AvailableBalance
// at some point if it makes senses overall the codebase
func (n marginUpdate) GeneralBalance() uint64 {
	var gen, bond uint64
	if n.general != nil {
		gen = n.general.Balance
	}
	if n.bond != nil {
		bond = n.bond.Balance
	}
	return bond + gen
}
