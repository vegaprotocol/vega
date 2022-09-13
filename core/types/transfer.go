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

type TransferInstruction struct {
	Owner     string
	Amount    *FinancialAmount
	Type      TransferInstructionType
	MinAmount *num.Uint
	Market    string
}

func (t *TransferInstruction) Clone() *TransferInstruction {
	cpy := *t
	cpy.Amount = t.Amount.Clone()
	cpy.MinAmount = t.MinAmount.Clone()
	return &cpy
}

// Merge creates a new Transfer.
func (t *TransferInstruction) Merge(oth *TransferInstruction) *TransferInstruction {
	if t.Owner != oth.Owner {
		panic(fmt.Sprintf("invalid transfer instruction merge, different owner specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	if t.Amount.Asset != oth.Amount.Asset {
		panic(fmt.Sprintf("invalid transfer instruction merge, different assets specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	if t.Type != oth.Type {
		panic(fmt.Sprintf("invalid transfer instruction merge, different types specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	if t.Market != oth.Market {
		panic(fmt.Sprintf("invalid transfer instruction merge, different markets specified, this should never happen: %v, %v", t.String(), oth.String()))
	}

	return &TransferInstruction{
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

func (t *TransferInstruction) IntoProto() *proto.TransferInstruction {
	return &proto.TransferInstruction{
		Owner:     t.Owner,
		Amount:    t.Amount.IntoProto(),
		Type:      t.Type,
		MinAmount: num.UintToString(t.MinAmount),
		MarketId:  t.Market,
	}
}

func TransferFromProto(p *proto.TransferInstruction) (*TransferInstruction, error) {
	amount, err := FinancialAmountFromProto(p.Amount)
	if err != nil {
		return nil, err
	}

	minAmount, overflow := num.UintFromString(p.MinAmount, 10)
	if overflow {
		return nil, errors.New("invalid min amount")
	}

	return &TransferInstruction{
		Owner:     p.Owner,
		Amount:    amount,
		Type:      p.Type,
		MinAmount: minAmount,
		Market:    p.MarketId,
	}, nil
}

func (t *TransferInstruction) String() string {
	return fmt.Sprintf(
		"owner(%s) amount(%s) type(%s) minAmount(%s)",
		t.Owner,
		reflectPointerToString(t.Amount),
		t.Type.String(),
		uintPointerToString(t.MinAmount),
	)
}

type TransferInstructionType = proto.TransferInstructionType

const (
	// Default value, always invalid.
	TransferInstructionTypeUnspecified TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_UNSPECIFIED
	// Loss.
	TransferInstructionTypeLoss TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_LOSS
	// Win.
	TransferInstructionTypeWin TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_WIN
	// Close.
	TransferInstructionTypeClose TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_CLOSE
	// Mark to market loss.
	TransferInstructionTypeMTMLoss TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MTM_LOSS
	// Mark to market win.
	TransferInstructionTypeMTMWin TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MTM_WIN
	// Margin too low.
	TransferInstructionTypeMarginLow TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MARGIN_LOW
	// Margin too high.
	TransferInstructionTypeMarginHigh TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MARGIN_HIGH
	// Margin was confiscated.
	TransferInstructionTypeMarginConfiscated TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MARGIN_CONFISCATED
	// Pay maker fee.
	TransferInstructionTypeMakerFeePay TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MAKER_FEE_PAY
	// Receive maker fee.
	TransferInstructionTypeMakerFeeReceive TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_MAKER_FEE_RECEIVE
	// Pay infrastructure fee.
	TransferInstructionTypeInfrastructureFeePay TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_INFRASTRUCTURE_FEE_PAY
	// Receive infrastructure fee.
	TransferInstructionTypeInfrastructureFeeDistribute TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE
	// Pay liquidity fee.
	TransferInstructionTypeLiquidityFeePay TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_LIQUIDITY_FEE_PAY
	// Receive liquidity fee.
	TransferInstructionTypeLiquidityFeeDistribute TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_LIQUIDITY_FEE_DISTRIBUTE
	// Bond too low.
	TransferInstructionTypeBondLow TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_BOND_LOW
	// Bond too high.
	TransferInstructionTypeBondHigh TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_BOND_HIGH
	// Lock amount for withdraw.
	TransferInstructionTypeWithdrawLock TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_WITHDRAW_LOCK
	// Actual withdraw from system.
	TransferInstructionTypeWithdraw TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_WITHDRAW
	// Deposit funds.
	TransferInstructionTypeDeposit TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_DEPOSIT
	// Bond slashing.
	TransferInstructionTypeBondSlashing TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_BOND_SLASHING
	// Stake reward.
	TransferInstructionTypeRewardPayout            TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_STAKE_REWARD
	TransferInstructionTypeTransferFundsSend       TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_TRANSFER_FUNDS_SEND
	TransferInstructionTypeTransferFundsDistribute TransferInstructionType = proto.TransferInstructionType_TRANSFER_INSTRUCTION_TYPE_TRANSFER_FUNDS_DISTRIBUTE
)
