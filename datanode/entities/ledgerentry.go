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

package entities

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/jackc/pgtype"
	"github.com/shopspring/decimal"
)

type LedgerEntry struct {
	LedgerEntryTime    time.Time
	FromAccountID      AccountID `db:"account_from_id"`
	ToAccountID        AccountID `db:"account_to_id"`
	Quantity           decimal.Decimal
	TxHash             TxHash
	VegaTime           time.Time
	TransferTime       time.Time
	Type               LedgerMovementType
	FromAccountBalance decimal.Decimal `db:"account_from_balance"`
	ToAccountBalance   decimal.Decimal `db:"account_to_balance"`
	TransferID         TransferID
}

var LedgerEntryColumns = []string{
	"ledger_entry_time",
	"account_from_id", "account_to_id", "quantity",
	"tx_hash", "vega_time", "transfer_time", "type",
	"account_from_balance",
	"account_to_balance",
	"transfer_id",
}

func (le LedgerEntry) ToProto(ctx context.Context, accountSource AccountSource) (*vega.LedgerEntry, error) {
	fromAcc, err := accountSource.GetByID(ctx, le.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting from account for transfer proto:%w", err)
	}

	toAcc, err := accountSource.GetByID(ctx, le.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting to account for transfer proto:%w", err)
	}

	var transferID *string
	if le.TransferID != "" {
		transferID = ptr.From(le.TransferID.String())
	}

	return &vega.LedgerEntry{
		FromAccount:        fromAcc.ToAccountDetailsProto(),
		ToAccount:          toAcc.ToAccountDetailsProto(),
		Amount:             le.Quantity.String(),
		Type:               vega.TransferType(le.Type),
		FromAccountBalance: le.FromAccountBalance.String(),
		ToAccountBalance:   le.ToAccountBalance.String(),
		TransferId:         transferID,
	}, nil
}

func (le LedgerEntry) ToRow() []any {
	return []any{
		le.LedgerEntryTime,
		le.FromAccountID,
		le.ToAccountID,
		le.Quantity,
		le.TxHash,
		le.VegaTime,
		le.TransferTime,
		le.Type,
		le.FromAccountBalance,
		le.ToAccountBalance,
		le.TransferID,
	}
}

func CreateLedgerEntryTime(vegaTime time.Time, seqNum int) time.Time {
	return vegaTime.Add(time.Duration(seqNum) * time.Microsecond)
}

type LedgerMovementType vega.TransferType

const (
	LedgerMovementTypeUnspecified                 = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_UNSPECIFIED)
	LedgerMovementTypeLoss                        = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_LOSS)
	LedgerMovementTypeWin                         = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_WIN)
	LedgerMovementTypeMTMLoss                     = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MTM_LOSS)
	LedgerMovementTypeMTMWin                      = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MTM_WIN)
	LedgerMovementTypeMarginLow                   = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MARGIN_LOW)
	LedgerMovementTypeMarginHigh                  = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MARGIN_HIGH)
	LedgerMovementTypeMarginConfiscated           = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MARGIN_CONFISCATED)
	LedgerMovementTypeMakerFeePay                 = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MAKER_FEE_PAY)
	LedgerMovementTypeMakerFeeReceive             = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE)
	LedgerMovementTypeInfrastructureFeePay        = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY)
	LedgerMovementTypeInfrastructureFeeDistribute = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE)
	LedgerMovementTypeLiquidityFeePay             = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY)
	LedgerMovementTypeLiquidityFeeDistribute      = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE)
	LedgerMovementTypeBondLow                     = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_BOND_LOW)
	LedgerMovementTypeBondHigh                    = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_BOND_HIGH)
	LedgerMovementTypeWithdraw                    = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_WITHDRAW)
	LedgerMovementTypeDeposit                     = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_DEPOSIT)
	LedgerMovementTypeBondSlashing                = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_BOND_SLASHING)
	LedgerMovementTypeRewardPayout                = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_REWARD_PAYOUT)
	LedgerMovementTypeTransferFundsSend           = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_TRANSFER_FUNDS_SEND)
	LedgerMovementTypeTransferFundsDistribute     = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_TRANSFER_FUNDS_DISTRIBUTE)
	LedgerMovementTypeClearAccount                = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_CLEAR_ACCOUNT)
	LedgerMovementTypePerpFundingWin              = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_PERPETUALS_FUNDING_WIN)
	LedgerMovementTypePerpFundingLoss             = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS)
	LedgerMovementTypeRewardsVested               = LedgerMovementType(vega.TransferType_TRANSFER_TYPE_REWARDS_VESTED)
)

func (l LedgerMovementType) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	ty, ok := vega.TransferType_name[int32(l)]
	if !ok {
		return buf, fmt.Errorf("unknown ledger movement type: %s", ty)
	}
	return append(buf, []byte(ty)...), nil
}

func (l *LedgerMovementType) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.TransferType_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown ledger movement type: %s", src)
	}

	*l = LedgerMovementType(val)
	return nil
}
