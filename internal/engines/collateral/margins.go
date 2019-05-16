package collateral

import types "code.vegaprotocol.io/vega/proto"

type marginUpdate struct {
	*types.Transfer
	margin  *types.Account
	general *types.Account
}

// Party - returns trader ID, part of the MarketPosition interface
func (m marginUpdate) Party() string {
	return m.Owner
}

// Size - returns multiplication factor for amount transferred, is either 1 or -1
// part of the MarketPosition interface
func (m marginUpdate) Size() int64 {
	s := int64(m.Transfer.Size)
	// losses are negative...
	if m.Type == types.TransferType_MTM_LOSS || m.Type == types.TransferType_LOSS {
		s *= -1
	}
	return s
}

// Price - returns the amount that was transferred, part of MarketPosition interface
func (m marginUpdate) Price() uint64 {
	// multiplied by size (either 1 or -1) -> yields absolute value, so we can safely return as uint64
	a := m.Size() * m.Transfer.Amount.Amount
	return uint64(a)
}

// MarginBalance - returns current balance of margin account
func (m marginUpdate) MarginBalance() uint64 {
	return uint64(m.margin.Balance)
}

// GeneralBalance - returns current balance of general account
func (m marginUpdate) GeneralBalance() uint64 {
	return uint64(m.general.Balance)
}
