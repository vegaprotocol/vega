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

package vault

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type RedeemStatus = vega.RedeemStatus

const (
	RedeemStatusUnspecified RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_UNSPECIFIED
	RedeemStatusPending     RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_PENDING
	RedeemStatusLate        RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_LATE
	RedeemStatusCompleted   RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_COMPLETED
)

type RedeemRequest struct {
	party     string
	date      time.Time
	amount    *num.Uint
	remaining num.Decimal
	status    RedeemStatus
}

type VaultState struct {
	log                     *logging.Logger
	vault                   *types.Vault
	shareHolders            map[string]num.Decimal
	collateral              Collateral
	broker                  Broker
	highWaterMark           num.Decimal
	investedAmount          *num.Uint
	nextFeeCalc             time.Time
	nextRedemptionDateIndex int
	redeemQueue             []*RedeemRequest
	lateRedemptions         []*RedeemRequest
	status                  types.VaultStatus
}

// NewVaultState creates a new vaults.
func NewVaultState(log *logging.Logger, vault *types.Vault, collateral Collateral, now time.Time, broker Broker) *VaultState {
	return &VaultState{
		log:                     log,
		collateral:              collateral,
		broker:                  broker,
		vault:                   vault,
		shareHolders:            map[string]num.Decimal{},
		highWaterMark:           num.DecimalOne(),
		investedAmount:          num.UintZero(),
		nextFeeCalc:             now.Add(vault.FeePeriod),
		nextRedemptionDateIndex: 0,
		status:                  types.VaultStatusActive,
	}
}

func (vs *VaultState) processWithdrawals(ctx context.Context, now time.Time) {
	if vs.nextRedemptionDateIndex >= len(vs.vault.RedemptionDates) {
		return
	}
	if !now.After(vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].RedemptionDate) {
		return
	}

	vaultBalance, _ := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	vaultLiquidBalance, _ := vs.collateral.GetVaultLiquidBalance(vs.vault.ID, vs.vault.Asset)

	if vaultBalance.IsZero() && vaultLiquidBalance.IsZero() {
		return
	}

	// if this is the last redemption date redeem for all parties
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates)-1 {
		vs.redeemQueue = []*RedeemRequest{}
		for party := range vs.shareHolders {
			vs.redeemQueue = append(vs.redeemQueue, &RedeemRequest{
				party:     party,
				date:      now,
				amount:    num.UintZero(),
				remaining: num.DecimalZero(),
				status:    RedeemStatusPending,
			})
		}
		vs.status = types.VaultStatusStopping
	}

	redeemRequests := []*RedeemRequest{}

	nextRedeemQueueIndex := 0
	for _, rr := range vs.redeemQueue {
		if !rr.date.Before(now) {
			redeemRequests = append(redeemRequests, rr)
			nextRedeemQueueIndex += 1
		} else {
			break
		}
	}
	maxFraction := vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].MaxFraction
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates)-1 {
		maxFraction = num.DecimalOne()
	}

	partyToRedeemed := map[string]num.Decimal{}
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates)-1 || vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].RedemptionType == types.RedemptionTypeNormal {
		availableAmount := maxFraction.Mul(vaultBalance.ToDecimal())
		actualLiquidAmount := maxFraction.Mul(vaultLiquidBalance.ToDecimal())

		for _, rr := range redeemRequests {
			requestedAmount := rr.amount
			partyShare := vs.shareHolders[rr.party].Mul(actualLiquidAmount)
			if rr.amount.IsZero() {
				rr.amount, _ = num.UintFromDecimal(partyShare)
				rr.remaining = partyShare
			}
			rr.remaining = num.MinD(rr.remaining, vs.shareHolders[rr.party].Mul(availableAmount))
			alreadyRedeemedThisRound := num.DecimalZero()
			if amt, ok := partyToRedeemed[rr.party]; ok {
				alreadyRedeemedThisRound = amt
			} else {
				partyToRedeemed[rr.party] = num.DecimalZero()
			}
			redeem := num.MinD(partyShare, requestedAmount.ToDecimal())
			alreadyRedeemedThisRound = num.MinD(alreadyRedeemedThisRound.Add(redeem), partyShare)
			rr.remaining = rr.remaining.Sub(alreadyRedeemedThisRound.Sub(partyToRedeemed[rr.party]))
			partyToRedeemed[rr.party] = alreadyRedeemedThisRound

			if rr.remaining.IsZero() {
				rr.status = RedeemStatusCompleted
			} else {
				rr.status = RedeemStatusLate
				vs.lateRedemptions = append(vs.lateRedemptions, rr)
			}
		}
	} else {
		availableAmount := vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].MaxFraction.Mul(vaultLiquidBalance.ToDecimal())
		for _, rr := range redeemRequests {
			requestedAmount := rr.amount
			partyShare := vs.shareHolders[rr.party].Mul(availableAmount)
			alreadyRedeemedThisRound := num.DecimalZero()
			if amt, ok := partyToRedeemed[rr.party]; ok {
				alreadyRedeemedThisRound = amt
			}
			redeem := num.MinD(partyShare, requestedAmount.ToDecimal())
			alreadyRedeemedThisRound = num.MinD(alreadyRedeemedThisRound.Add(redeem), partyShare)
			partyToRedeemed[rr.party] = alreadyRedeemedThisRound
			rr.status = RedeemStatusCompleted
		}
	}
	vs.redeemQueue = vs.redeemQueue[nextRedeemQueueIndex:]
	vs.processLiquidRedemptions(ctx, partyToRedeemed)
	vs.nextRedemptionDateIndex += 1
}

func (vs *VaultState) processLiquidRedemptions(ctx context.Context, partyToAmount map[string]num.Decimal) {
	keys := make([]string, 0, len(partyToAmount))
	for k := range partyToAmount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vaultBalance, err := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	if err != nil {
		vs.log.Panic("failed to get vault balance", logging.Error(err))
	}
	totalRedeemed := num.UintZero()
	for _, party := range keys {
		amount, _ := num.UintFromDecimal(partyToAmount[party])
		if err := vs.updateSharesOnRedeem(ctx, vaultBalance, party, amount); err != nil {
			vs.log.Panic("failed to update vault on redemption")
		}
		vaultBalance.Sub(vaultBalance, amount)
		totalRedeemed.AddSum(amount)
	}
}

func (vs *VaultState) updateSharesOnRedeem(ctx context.Context, vaultBalance *num.Uint, party string, amount *num.Uint) error {
	le, err := vs.collateral.WithdrawFromVault(ctx, vs.vault.ID, vs.vault.Asset, party, amount)
	if err != nil {
		return err
	}
	if le == nil {
		return fmt.Errorf("failed to redeem from vault")
	}
	vs.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{le}))

	newBalanceD := num.UintZero().Sub(vaultBalance, amount).ToDecimal()
	newShare := make(map[string]num.Decimal, len(vs.shareHolders))
	shareChange := num.DecimalZero()
	for vParty, share := range vs.shareHolders {
		if vParty != party {
			newShare[vParty] = share.Mul(vaultBalance.ToDecimal()).Div(newBalanceD)
		} else {
			newShare[vParty] = (share.Mul(vaultBalance.ToDecimal()).Add(amount.ToDecimal())).Div(newBalanceD)
			shareChange = share.Sub(newShare[vParty])
		}
	}
	vs.shareHolders = newShare
	vs.investedAmount, _ = num.UintFromDecimal(vs.investedAmount.ToDecimal().Mul(num.DecimalOne().Sub(shareChange)))
	return nil
}

func (vs *VaultState) processLateRedemptions(ctx context.Context) {
	remainingLateRedemptions := []*RedeemRequest{}
	if len(vs.lateRedemptions) == 0 {
		return
	}
	vaultLiquidBalance, _ := vs.collateral.GetVaultLiquidBalance(vs.vault.ID, vs.vault.Asset)
	vaultLiquidBalanceD := vaultLiquidBalance.ToDecimal()
	partyToRedeemed := map[string]num.Decimal{}
	for _, rr := range vs.lateRedemptions {
		requestedAmount := rr.remaining
		partyShare := vs.shareHolders[rr.party].Mul(vaultLiquidBalanceD)
		alreadyRedeemedThisRound := num.DecimalZero()
		if amt, ok := partyToRedeemed[rr.party]; ok {
			alreadyRedeemedThisRound = amt
		} else {
			partyToRedeemed[rr.party] = num.DecimalZero()
		}
		redeem := num.MinD(partyShare, requestedAmount)
		alreadyRedeemedThisRound = num.MinD(alreadyRedeemedThisRound.Add(redeem), partyShare)
		rr.remaining = rr.remaining.Sub(alreadyRedeemedThisRound.Sub(partyToRedeemed[rr.party]))
		partyToRedeemed[rr.party] = alreadyRedeemedThisRound
		if !rr.remaining.IsZero() {
			remainingLateRedemptions = append(remainingLateRedemptions, rr)
		}
	}
	vs.processLiquidRedemptions(ctx, partyToRedeemed)
	vs.lateRedemptions = remainingLateRedemptions
}

// WithdrawFromVault generate a new redeem request in the redeem queue with the time corresponding to now + the cutoff period (in days).
func (vs *VaultState) WithdrawFromVault(ctx context.Context, party string, amount *num.Uint, now time.Time) error {
	if vs.status != types.VaultStatusActive {
		return fmt.Errorf("vault is not active")
	}
	if _, ok := vs.shareHolders[party]; !ok {
		return fmt.Errorf("party has no share in the vault")
	}
	vs.redeemQueue = append(vs.redeemQueue, &RedeemRequest{
		party:     party,
		amount:    amount,
		remaining: amount.ToDecimal(),
		date:      now.Add(time.Hour * 24 * time.Duration(vs.vault.CutOffPeriodLength)),
		status:    RedeemStatusPending,
	})
	return nil
}

// ChangeOwner updates the public key of the owner of the vault.
func (vs *VaultState) ChangeOwner(ctx context.Context, currentOwner, newOwner string) error {
	if vs.vault.Owner != currentOwner {
		return fmt.Errorf("only the current owner of the vault can change ownership")
	}
	if vs.status != types.VaultStatusActive {
		return fmt.Errorf("vault is closed")
	}
	vs.vault.Owner = newOwner
	return nil
}

// UpdateVault updates the configuration of a vault if the new configuration is valid.
func (vs *VaultState) UpdateVault(vault *types.Vault, now time.Time, minNoticePeriodInDays int64) error {
	asset := vs.vault.Asset
	// if the first redemption date is in the past, reject the update
	if vault.RedemptionDates[0].RedemptionDate.Before(now) {
		return fmt.Errorf("redemptions dates are not allowed to be in the past")
	}

	// we expect all dates before the notice period to remain unchanged
	updatedIndex := 0
	for index := vs.nextRedemptionDateIndex; index < len(vs.vault.RedemptionDates); index++ {
		if !vault.RedemptionDates[index].RedemptionDate.Add(-time.Hour * 24 * time.Duration(minNoticePeriodInDays)).Before(now) {
			break
		}
		if !vs.vault.RedemptionDates[index].MaxFraction.Equal(vault.RedemptionDates[updatedIndex].MaxFraction) ||
			!vs.vault.RedemptionDates[index].RedemptionDate.Equal(vault.RedemptionDates[updatedIndex].RedemptionDate) ||
			vs.vault.RedemptionDates[index].RedemptionType != vault.RedemptionDates[updatedIndex].RedemptionType {
			return fmt.Errorf("redemption dates within notice period are not allowed to change")
		}
		updatedIndex += 1
	}

	// we expect the first date to remain unchanged even if it is after the notice period.
	// NB: this assumes that no redemption dates can be added *before* the next redemption date.
	if !vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].MaxFraction.Equal(vault.RedemptionDates[0].MaxFraction) ||
		!vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].RedemptionDate.Equal(vault.RedemptionDates[0].RedemptionDate) ||
		vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].RedemptionType != vault.RedemptionDates[0].RedemptionType {
		return fmt.Errorf("next redemption date is not allowed to change")
	}

	vs.vault = vault
	vs.vault.Asset = asset
	return nil
}

// DepositToVault transfer funds from the public key of the party to the vault. It updates the share holdings of all parties to reflect the new balance.
func (vs *VaultState) DepositToVault(ctx context.Context, party string, amount *num.Uint) error {
	if vs.status != types.VaultStatusActive {
		return fmt.Errorf("vault is not active")
	}
	// Get the total balance of the vault before the deposit
	vaultBalance, err := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	if err != nil {
		return err
	}

	// transfer the funds to the vault
	le, err := vs.collateral.DepositToVault(ctx, vs.vault.ID, vs.vault.Asset, party, amount)
	if err != nil {
		return err
	}
	if le == nil {
		return fmt.Errorf("failed to deposit to vault")
	}
	vs.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{le}))

	// update the shares in the vault
	newBalance := num.Sum(vaultBalance, amount).ToDecimal()
	newShare := make(map[string]num.Decimal, len(vs.shareHolders))
	if _, ok := vs.shareHolders[party]; !ok {
		vs.shareHolders[party] = num.DecimalZero()
	}
	for vParty, share := range vs.shareHolders {
		if vParty != party {
			newShare[vParty] = share.Mul(vaultBalance.ToDecimal()).Div(newBalance)
		} else {
			newShare[vParty] = (share.Mul(vaultBalance.ToDecimal()).Add(amount.ToDecimal())).Div(newBalance)
		}
	}

	// make sure that we don't have a rounding error that makes the total shares of the vault greater than 1
	VerifyAndCapAt1(newShare)
	vs.shareHolders = newShare

	// update the invested amount
	vs.investedAmount.AddSum(amount)
	return nil
}

// ProcessFees handles the collection of fees.
func (vs *VaultState) ProcessFees(now time.Time) {
	if vs.status != types.VaultStatusActive {
		return
	}
	vaultBalance, err := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	if err != nil {
		return
	}
	newHighWatermark := num.MaxD(vs.highWaterMark, vaultBalance.ToDecimal().Div(vs.investedAmount.ToDecimal()))
	newGains := num.MaxD(num.DecimalZero(), newHighWatermark.Sub(vs.highWaterMark)).Mul(vs.investedAmount.ToDecimal())
	vs.highWaterMark = newHighWatermark

	if len(vs.lateRedemptions) > 0 {
		// no fees if there are active late redemptions.
		return
	}

	performanceFee := vs.vault.PerformanceFeeFactor.Mul(newGains)
	managementFee := vs.vault.ManagementFeeFactor.Mul(vaultBalance.ToDecimal())
	totalFees := performanceFee.Add(managementFee)
	totalSharesNotOwner := num.DecimalZero()
	for vParty, share := range vs.shareHolders {
		if vParty != vs.vault.Owner {
			vs.shareHolders[vParty] = share.Mul(vaultBalance.ToDecimal().Sub(totalFees)).Div(vaultBalance.ToDecimal())
			totalSharesNotOwner = totalSharesNotOwner.Add(vs.shareHolders[vParty])
		}
	}
	vs.shareHolders[vs.vault.Owner] = num.DecimalOne().Sub(totalSharesNotOwner)
	vs.nextFeeCalc = now.Add(vs.vault.FeePeriod)
}

// GetVaultShares returns a copy of the current share holding of the vault.
func (vs *VaultState) GetVaultShares() map[string]num.Decimal {
	shareHolders := make(map[string]num.Decimal, len(vs.shareHolders))
	for party, share := range vs.shareHolders {
		shareHolders[party] = share
	}
	return shareHolders
}

// VerifyAndCapAt1 is a utility that ensures that decimal rounding error gets the total share holding above one. If it
// ever does get above one it is capped by correcting the max share.
func VerifyAndCapAt1(shares map[string]num.Decimal) {
	total := num.DecimalZero()
	maxParty := ""
	maxShare := num.DecimalZero()
	for party, share := range shares {
		if share.GreaterThan(maxShare) {
			maxShare = share
			maxParty = party
			total = total.Add(share)
		}
	}
	if total.LessThanOrEqual(num.DecimalOne()) {
		return
	}

	shares[maxParty] = maxShare.Sub(total.Sub(num.DecimalOne()))
}
