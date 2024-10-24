// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
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
	Owner      string
	Amount     *FinancialAmount
	Type       TransferType
	MinAmount  *num.Uint
	Market     string
	TransferID *string
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
		Type:       t.Type,
		MinAmount:  num.Sum(t.MinAmount, t.MinAmount),
		Market:     t.Market,
		TransferID: t.TransferID,
	}
}

func (f FinancialAmount) String() string {
	return fmt.Sprintf(
		"asset(%s) amount(%s)",
		f.Asset,
		stringer.PtrToString(f.Amount),
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
		stringer.PtrToString(t.Amount),
		t.Type.String(),
		stringer.PtrToString(t.MinAmount),
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
	TransferTypeRewardPayout               TransferType = proto.TransferType_TRANSFER_TYPE_REWARD_PAYOUT
	TransferTypeTransferFundsSend          TransferType = proto.TransferType_TRANSFER_TYPE_TRANSFER_FUNDS_SEND
	TransferTypeTransferFundsDistribute    TransferType = proto.TransferType_TRANSFER_TYPE_TRANSFER_FUNDS_DISTRIBUTE
	TransferTypeClearAccount               TransferType = proto.TransferType_TRANSFER_TYPE_CLEAR_ACCOUNT
	TransferTypeCheckpointBalanceRestore   TransferType = proto.TransferType_TRANSFER_TYPE_CHECKPOINT_BALANCE_RESTORE
	TransferTypeSuccessorInsuranceFraction TransferType = proto.TransferType_TRANSFER_TYPE_SUCCESSOR_INSURANCE_FRACTION
	TransferTypeSpot                       TransferType = proto.TransferType_TRANSFER_TYPE_SPOT
	TransferTypeHoldingAccount             TransferType = proto.TransferType_TRANSFER_TYPE_HOLDING_LOCK
	TransferTypeReleaseHoldingAccount      TransferType = proto.TransferType_TRANSFER_TYPE_HOLDING_RELEASE
	// Liquidity fees.
	TransferTypeLiquidityFeeAllocate          TransferType = proto.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE
	TransferTypeLiquidityFeeNetDistribute     TransferType = proto.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_NET_DISTRIBUTE
	TransferTypeSLAPenaltyBondApply           TransferType = proto.TransferType_TRANSFER_TYPE_SLA_PENALTY_BOND_APPLY
	TransferTypeSLAPenaltyLpFeeApply          TransferType = proto.TransferType_TRANSFER_TYPE_SLA_PENALTY_LP_FEE_APPLY
	TransferTypeLiquidityFeeUnpaidCollect     TransferType = proto.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_UNPAID_COLLECT
	TransferTypeSlaPerformanceBonusDistribute TransferType = proto.TransferType_TRANSFER_TYPE_SLA_PERFORMANCE_BONUS_DISTRIBUTE
	// perps funding.
	TransferTypePerpFundingLoss TransferType = proto.TransferType_TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS
	TransferTypePerpFundingWin  TransferType = proto.TransferType_TRANSFER_TYPE_PERPETUALS_FUNDING_WIN
	TransferTypeRewardsVested   TransferType = proto.TransferType_TRANSFER_TYPE_REWARDS_VESTED

	TransferTypeFeeReferrerRewardPay        TransferType = proto.TransferType_TRANSFER_TYPE_FEE_REFERRER_REWARD_PAY
	TransferTypeFeeReferrerRewardDistribute TransferType = proto.TransferType_TRANSFER_TYPE_FEE_REFERRER_REWARD_DISTRIBUTE

	TransferTypeOrderMarginLow    TransferType = proto.TransferType_TRANSFER_TYPE_ORDER_MARGIN_LOW
	TransferTypeOrderMarginHigh   TransferType = proto.TransferType_TRANSFER_TYPE_ORDER_MARGIN_HIGH
	TransferTypeIsolatedMarginLow TransferType = proto.TransferType_TRANSFER_TYPE_ISOLATED_MARGIN_LOW
	// AMM account juggling.
	TransferTypeAMMLow     TransferType = proto.TransferType_TRANSFER_TYPE_AMM_LOW
	TransferTypeAMMHigh    TransferType = proto.TransferType_TRANSFER_TYPE_AMM_HIGH
	TransferTypeAMMRelease TransferType = proto.TransferType_TRANSFER_TYPE_AMM_RELEASE
	// additional fees transfers.
	TransferTypeBuyBackFeePay TransferType = proto.TransferType_TRANSFER_TYPE_BUY_BACK_FEE_PAY
	TransferTypeTreasuryPay   TransferType = proto.TransferType_TRANSFER_TYPE_TREASURY_FEE_PAY
	// Pay high maker fee.
	TransferTypeHighMakerRebatePay TransferType = proto.TransferType_TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_PAY
	// Receive high maker rebate.
	TransferTypeHighMakerRebateReceive TransferType = proto.TransferType_TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE
	// Deposit to vault.
	TransferTypeDepositToVault TransferType = proto.TransferType_TRANSFER_TYPE_DEPOSIT_TO_VAULT
	// Withdraw from vault.
	TransferTypeWithdrawFromVault TransferType = proto.TransferType_TRANSFER_TYPE_WITHDRAW_FROM_VAULT
)
