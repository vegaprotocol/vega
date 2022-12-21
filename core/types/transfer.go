// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type FinancialAmount struct {
	Asset  string
	Amount *num.Uint
}

func (f *FinancialAmount) Clone() *FinancialAmount {
	cpy := *f
	cpy.Amount = f.Amount.Clone()
	return &cpy
}

type Transfer struct {
	Owner     string
	Amount    *FinancialAmount
	Type      TransferType
	MinAmount *num.Uint
	Market    string
}

func (t *Transfer) Clone() *Transfer {
	cpy := *t
	cpy.Amount = t.Amount.Clone()
	cpy.MinAmount = t.MinAmount.Clone()
	return &cpy
}

// Merge creates a new Transfer.
func (t *Transfer) Merge(oth *Transfer) *Transfer {
	if t.Owner != oth.Owner {
		panic(fmt.Sprintf("invalid transfer merge, different owner specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	if t.Amount.Asset != oth.Amount.Asset {
		panic(fmt.Sprintf("invalid transfer merge, different assets specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	if t.Type != oth.Type {
		panic(fmt.Sprintf("invalid transfer merge, different types specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	if t.Market != oth.Market {
		panic(fmt.Sprintf("invalid transfer merge, different markets specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	return &Transfer{
		Owner: t.Owner,
		Amount: &FinancialAmount{
			Asset:  t.Amount.Asset,
			Amount: num.Sum(t.Amount.Amount, oth.Amount.Amount),
		},
		Type:      t.Type,
		MinAmount: num.Sum(t.MinAmount, t.MinAmount),
		Market:    t.Market,
	}
}

func (f FinancialAmount) String() string {
	return fmt.Sprintf(
		"asset(%s) amount(%s)",
		f.Asset,
		uintPointerToString(f.Amount),
	)
}

func (f *FinancialAmount) IntoProto() *proto.FinancialAmount {
	return &proto.FinancialAmount{
		Asset:  f.Asset,
		Amount: num.UintToString(f.Amount),
	}
}

func FinancialAmountFromProto(p *proto.FinancialAmount) (*FinancialAmount, error) {
	amount, overflow := num.UintFromString(p.Amount, 10)
	if overflow {
		return nil, errors.New("invalid amount")
	}

	return &FinancialAmount{
		Asset:  p.Asset,
		Amount: amount,
	}, nil
}

func (t *Transfer) IntoProto() *proto.Transfer {
	return &proto.Transfer{
		Owner:     t.Owner,
		Amount:    t.Amount.IntoProto(),
		Type:      t.Type,
		MinAmount: num.UintToString(t.MinAmount),
		MarketId:  t.Market,
	}
}

func TransferFromProto(p *proto.Transfer) (*Transfer, error) {
	amount, err := FinancialAmountFromProto(p.Amount)
	if err != nil {
		return nil, err
	}

	minAmount, overflow := num.UintFromString(p.MinAmount, 10)
	if overflow {
		return nil, errors.New("invalid min amount")
	}

	return &Transfer{
		Owner:     p.Owner,
		Amount:    amount,
		Type:      p.Type,
		MinAmount: minAmount,
		Market:    p.MarketId,
	}, nil
}

func (t *Transfer) String() string {
	return fmt.Sprintf(
		"owner(%s) amount(%s) type(%s) minAmount(%s)",
		t.Owner,
		reflectPointerToString(t.Amount),
		t.Type.String(),
		uintPointerToString(t.MinAmount),
	)
}

type TransferType = proto.TransferType

const (
	// Default value, always invalid.
	TransferTypeUnspecified TransferType = proto.TransferType_TRANSFER_TYPE_UNSPECIFIED
	// Loss.
	TransferTypeLoss TransferType = proto.TransferType_TRANSFER_TYPE_LOSS
	// Win.
	TransferTypeWin TransferType = proto.TransferType_TRANSFER_TYPE_WIN
	// Mark to market loss.
	TransferTypeMTMLoss TransferType = proto.TransferType_TRANSFER_TYPE_MTM_LOSS
	// Mark to market win.
	TransferTypeMTMWin TransferType = proto.TransferType_TRANSFER_TYPE_MTM_WIN
	// Margin too low.
	TransferTypeMarginLow TransferType = proto.TransferType_TRANSFER_TYPE_MARGIN_LOW
	// Margin too high.
	TransferTypeMarginHigh TransferType = proto.TransferType_TRANSFER_TYPE_MARGIN_HIGH
	// Margin was confiscated.
	TransferTypeMarginConfiscated TransferType = proto.TransferType_TRANSFER_TYPE_MARGIN_CONFISCATED
	// Pay maker fee.
	TransferTypeMakerFeePay TransferType = proto.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY
	// Receive maker fee.
	TransferTypeMakerFeeReceive TransferType = proto.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE
	// Pay infrastructure fee.
	TransferTypeInfrastructureFeePay TransferType = proto.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY
	// Receive infrastructure fee.
	TransferTypeInfrastructureFeeDistribute TransferType = proto.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE
	// Pay liquidity fee.
	TransferTypeLiquidityFeePay TransferType = proto.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY
	// Receive liquidity fee.
	TransferTypeLiquidityFeeDistribute TransferType = proto.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE
	// Bond too low.
	TransferTypeBondLow TransferType = proto.TransferType_TRANSFER_TYPE_BOND_LOW
	// Bond too high.
	TransferTypeBondHigh TransferType = proto.TransferType_TRANSFER_TYPE_BOND_HIGH
	// Actual withdraw from system.
	TransferTypeWithdraw TransferType = proto.TransferType_TRANSFER_TYPE_WITHDRAW
	// Deposit funds.
	TransferTypeDeposit TransferType = proto.TransferType_TRANSFER_TYPE_DEPOSIT
	// Bond slashing.
	TransferTypeBondSlashing TransferType = proto.TransferType_TRANSFER_TYPE_BOND_SLASHING
	// Reward payout.
	TransferTypeRewardPayout             TransferType = proto.TransferType_TRANSFER_TYPE_REWARD_PAYOUT
	TransferTypeTransferFundsSend        TransferType = proto.TransferType_TRANSFER_TYPE_TRANSFER_FUNDS_SEND
	TransferTypeTransferFundsDistribute  TransferType = proto.TransferType_TRANSFER_TYPE_TRANSFER_FUNDS_DISTRIBUTE
	TransferTypeClearAccount             TransferType = proto.TransferType_TRANSFER_TYPE_CLEAR_ACCOUNT
	TransferTypeCheckpointBalanceRestore TransferType = proto.TransferType_TRANSFER_TYPE_CHECKPOINT_BALANCE_RESTORE
)
