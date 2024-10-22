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

package banking

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// EstimateFee returns a transaction fee estimate with potential discount that can be applied to it.
func EstimateFee(
	assetQuantum num.Decimal,
	maxQuantumAmount num.Decimal,
	transferFeeFactor num.Decimal,
	amount *num.Uint,
	accumulatedDiscount *num.Uint,
	from string,
	fromAccountType types.AccountType,
	fromDerivedKey *string,
	to string,
	toAccountType types.AccountType,
) (fee *num.Uint, discount *num.Uint) {
	tFee := calculateFeeForTransfer(assetQuantum, maxQuantumAmount, transferFeeFactor, amount, from,
		fromAccountType, fromDerivedKey, to, toAccountType)
	return calculateDiscount(accumulatedDiscount, tFee)
}

func calculateFeeForTransfer(
	assetQuantum num.Decimal,
	maxQuantumAmount num.Decimal,
	transferFeeFactor num.Decimal,
	amount *num.Uint,
	from string,
	fromAccountType types.AccountType,
	fromDerivedKey *string,
	to string,
	toAccountType types.AccountType,
) *num.Uint {
	feeAmount := num.UintZero()

	// no fee for Vested account
	// either from owner's vested to their general account or from derived key vested to owner's general account
	if (fromAccountType == types.AccountTypeVestedRewards && (from == to || fromDerivedKey != nil)) ||
		(fromAccountType == types.AccountTypeLockedForStaking && from == to) ||
		(fromAccountType == types.AccountTypeGeneral && toAccountType == types.AccountTypeLockedForStaking && from == to) {
		return feeAmount
	}

	// now we calculate the fee
	// min(transfer amount * transfer.fee.factor, transfer.fee.maxQuantumAmount * quantum)
	feeAmount, _ = num.UintFromDecimal(num.MinD(
		amount.ToDecimal().Mul(transferFeeFactor),
		maxQuantumAmount.Mul(assetQuantum),
	))

	return feeAmount
}
