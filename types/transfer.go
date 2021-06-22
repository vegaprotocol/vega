package types

import (
	"code.vegaprotocol.io/vega/proto"
)

func (f FinancialAmount) String() string {
	return f.IntoProto().String()
}

func (f *FinancialAmount) IntoProto() *proto.FinancialAmount {
	return &proto.FinancialAmount{
		Asset:  f.Asset,
		Amount: f.Amount.Uint64(),
	}
}

func (t *Transfer) IntoProto() *proto.Transfer {
	p := &proto.Transfer{
		Owner:  t.Owner,
		Amount: t.Amount.IntoProto(),
		Type:   t.Type,
	}
	if t.MinAmount != nil {
		p.MinAmount = t.MinAmount.Uint64()
	}
	return p
}

func (t *Transfer) String() string {
	return t.IntoProto().String()
}

type TransferType = proto.TransferType

const (
	// Default value, always invalid
	TransferType_TRANSFER_TYPE_UNSPECIFIED TransferType = 0
	// Loss
	TransferType_TRANSFER_TYPE_LOSS TransferType = 1
	// Win
	TransferType_TRANSFER_TYPE_WIN TransferType = 2
	// Close
	TransferType_TRANSFER_TYPE_CLOSE TransferType = 3
	// Mark to market loss
	TransferType_TRANSFER_TYPE_MTM_LOSS TransferType = 4
	// Mark to market win
	TransferType_TRANSFER_TYPE_MTM_WIN TransferType = 5
	// Margin too low
	TransferType_TRANSFER_TYPE_MARGIN_LOW TransferType = 6
	// Margin too high
	TransferType_TRANSFER_TYPE_MARGIN_HIGH TransferType = 7
	// Margin was confiscated
	TransferType_TRANSFER_TYPE_MARGIN_CONFISCATED TransferType = 8
	// Pay maker fee
	TransferType_TRANSFER_TYPE_MAKER_FEE_PAY TransferType = 9
	// Receive maker fee
	TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE TransferType = 10
	// Pay infrastructure fee
	TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY TransferType = 11
	// Receive infrastructure fee
	TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE TransferType = 12
	// Pay liquidity fee
	TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY TransferType = 13
	// Receive liquidity fee
	TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE TransferType = 14
	// Bond too low
	TransferType_TRANSFER_TYPE_BOND_LOW TransferType = 15
	// Bond too high
	TransferType_TRANSFER_TYPE_BOND_HIGH TransferType = 16
	// Lock amount for withdraw
	TransferType_TRANSFER_TYPE_WITHDRAW_LOCK TransferType = 17
	// Actual withdraw from system
	TransferType_TRANSFER_TYPE_WITHDRAW TransferType = 18
	// Deposit funds
	TransferType_TRANSFER_TYPE_DEPOSIT TransferType = 19
	// Bond slashing
	TransferType_TRANSFER_TYPE_BOND_SLASHING TransferType = 20
)
