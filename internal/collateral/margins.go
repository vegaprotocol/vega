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
