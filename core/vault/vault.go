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
)

type RedeemRequest struct {
	Party     string
	Date      time.Time
	Amount    *num.Uint
	Remaining *num.Uint
	Status    types.RedeemStatus
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

func (vs *VaultState) ProcessWithdrawals(ctx context.Context, now time.Time) {
	// the vault is in stopping mode - nothing to do here
	if vs.nextRedemptionDateIndex >= len(vs.vault.RedemptionDates) {
		return
	}

	// if we're not yet reached the next redemption date - nothing to do here
	if now.Before(vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].RedemptionDate) {
		return
	}

	// get all redeem request that are past the cutoff for the given redemption date
	redeemRequests := vs.GetRedemptionRequestForDate(now)

	// on the last redemption date the
	maxFraction := vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].MaxFraction
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates)-1 {
		maxFraction = num.DecimalOne()
	}

	// calculate the available balances
	vaultBalance, _ := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	vaultLiquidBalance, _ := vs.collateral.GetVaultLiquidBalance(vs.vault.ID, vs.vault.Asset)

	// if this is the last redemption date - the type is automatically normal regardless of what it was setup with
	// as everything needs to go
	redemptionType := vs.vault.RedemptionDates[vs.nextRedemptionDateIndex].RedemptionType
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates)-1 {
		redemptionType = types.RedemptionTypeNormal
	}
	partyToRedeemed, lateRedemptions := PrepareRedemptions(vs.shareHolders, redeemRequests, vaultBalance, vaultLiquidBalance, redemptionType, maxFraction)

	// add late redemptions
	vs.lateRedemptions = append(vs.lateRedemptions, lateRedemptions...)

	// remove completed redemptions from the queue
	vs.redeemQueue = vs.redeemQueue[len(redeemRequests):]

	// process redemption transfers and update the shares
	vs.redeem(ctx, partyToRedeemed)

	// progress index to the next redemption date
	vs.nextRedemptionDateIndex += 1
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates) {
		if len(vs.lateRedemptions) == 0 {
			vs.status = types.VaultStatusStopped
			finalBalance, _ := vs.collateral.GetVaultLiquidBalance(vs.vault.ID, vs.vault.Asset)
			if !finalBalance.IsZero() {
				le, err := vs.collateral.WithdrawFromVault(ctx, vs.vault.ID, vs.vault.Asset, vs.vault.Owner, finalBalance)
				if err == nil && le != nil {
					vs.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{le}))
				}
				vs.collateral.CloseVaultAccount(ctx, vs.vault.ID)
			}
		} else {
			vs.status = types.VaultStatusStopping
		}
	}
}

func (vs *VaultState) redeem(ctx context.Context, partyToAmount map[string]*num.Uint) {
	keys := make([]string, 0, len(partyToAmount))
	for k := range partyToAmount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vaultBalance, err := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	if err != nil {
		vs.log.Panic("failed to get vault balance", logging.Error(err))
	}
	for _, party := range keys {
		amount := partyToAmount[party]
		if err := vs.UpdateSharesOnRedeem(ctx, vaultBalance, party, amount); err != nil {
			vs.log.Panic("failed to update vault on redemption")
		}
		vaultBalance.Sub(vaultBalance, amount)
	}
}

func (vs *VaultState) ProcessLateRedemptions(ctx context.Context) {
	remainingLateRedemptions := []*RedeemRequest{}
	if len(vs.lateRedemptions) == 0 {
		return
	}
	vaultBalance, _ := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
	vaultBalanceD := vaultBalance.ToDecimal()
	vaultLiquidBalance, _ := vs.collateral.GetVaultLiquidBalance(vs.vault.ID, vs.vault.Asset)
	vaultLiquidBalanceD := vaultLiquidBalance.ToDecimal()

	if vaultLiquidBalance.IsZero() {
		return
	}
	partyToRedeemed := map[string]*num.Uint{}
	for _, rr := range vs.lateRedemptions {
		var requestedAmount *num.Uint
		partyShareOfLiquid, _ := num.UintFromDecimal(vs.shareHolders[rr.Party].Mul(vaultLiquidBalanceD))
		partyShareOfTotal, _ := num.UintFromDecimal(vs.shareHolders[rr.Party].Mul(vaultBalanceD))
		if rr.Remaining.IsZero() {
			requestedAmount = partyShareOfTotal
		} else {
			requestedAmount = rr.Remaining.Clone()
		}

		alreadyRedeemedThisRound := num.UintZero()
		if amt, ok := partyToRedeemed[rr.Party]; ok {
			alreadyRedeemedThisRound = amt
		} else {
			partyToRedeemed[rr.Party] = num.UintZero()
		}
		redeem := num.Min(num.UintZero().Sub(partyShareOfLiquid, alreadyRedeemedThisRound), requestedAmount)
		if !rr.Remaining.IsZero() {
			rr.Remaining = num.UintZero().Sub(rr.Remaining, redeem)
		}
		alreadyRedeemedThisRound.AddSum(redeem)
		partyToRedeemed[rr.Party] = alreadyRedeemedThisRound
		if !rr.Remaining.IsZero() {
			remainingLateRedemptions = append(remainingLateRedemptions, rr)
		}
	}
	vs.redeem(ctx, partyToRedeemed)
	vs.lateRedemptions = remainingLateRedemptions
}

// WithdrawFromVault generate a new redeem request in the redeem queue with the time corresponding to now + the cutoff period (in days).
func (vs *VaultState) WithdrawFromVault(ctx context.Context, party string, amount *num.Uint, now time.Time) error {
	if vs.status != types.VaultStatusActive {
		return fmt.Errorf("vault is not active")
	}
	if share, ok := vs.shareHolders[party]; !ok {
		return fmt.Errorf("party has no share in the vault")
	} else {
		vaultBalance, err := vs.collateral.GetVaultBalance(vs.vault.ID, vs.vault.Asset)
		if err != nil {
			return err
		}
		totalAvailableAmount, _ := num.UintFromDecimal(vaultBalance.ToDecimal().Mul(share))
		if amount.GT(totalAvailableAmount) {
			return fmt.Errorf("requested amount is greater than the share available")
		}
	}
	vs.redeemQueue = append(vs.redeemQueue, &RedeemRequest{
		Party:     party,
		Amount:    amount.Clone(),
		Remaining: amount.Clone(),
		Date:      now.Add(time.Hour * 24 * time.Duration(vs.vault.CutOffPeriodLength)),
		Status:    types.RedeemStatusPending,
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

// GetInvestmentTotal returns the current value of the investment amount.
func (vs *VaultState) GetInvestmentTotal() *num.Uint {
	return vs.investedAmount.Clone()
}

// UpdateSharesOnRedeem updates the vault on a single redemption for a given party. It transfers the balance and updaes the shares
// of the remaining share holders and the total invested amount.
func (vs *VaultState) UpdateSharesOnRedeem(ctx context.Context, vaultBalance *num.Uint, party string, amount *num.Uint) error {
	share, ok := vs.shareHolders[party]
	if !ok {
		vs.log.Panic("trying to update shares on redeem for party with no share")
	}

	impliedAmount, _ := num.UintFromDecimal(vaultBalance.ToDecimal().Mul(share))
	if impliedAmount.LT(amount) {
		vs.log.Panic("trying to withdraw more than the share")
	}
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
	for vParty, share := range vs.shareHolders {
		if vParty != party {
			newShare[vParty] = share.Mul(vaultBalance.ToDecimal()).Div(newBalanceD)
		} else {
			newShare[vParty] = (share.Mul(vaultBalance.ToDecimal()).Sub(amount.ToDecimal())).Div(newBalanceD)
		}
	}
	VerifyAndCapAt1(newShare)
	vs.shareHolders = newShare
	vs.investedAmount, _ = num.UintFromDecimal(vs.investedAmount.ToDecimal().Mul(num.DecimalOne().Sub(amount.ToDecimal().Div(vaultBalance.ToDecimal()))))
	return nil
}

// VerifyAndCapAt1 is a utility that ensures that decimal rounding error gets the total share holding above one. If it
// ever does get above one it is capped by correcting the max share.
func VerifyAndCapAt1(shares map[string]num.Decimal) {
	total := num.DecimalZero()
	maxParty := ""
	maxShare := num.DecimalZero()
	keys := make([]string, 0, len(shares))
	for k := range shares {
		keys = append(keys, k)
	}
	for _, party := range keys {
		share := shares[party]
		if share.GreaterThan(maxShare) {
			maxShare = share
			maxParty = party
		}
		total = total.Add(share)
	}
	if total.LessThanOrEqual(num.DecimalOne()) {
		return
	}

	shares[maxParty] = maxShare.Sub(total.Sub(num.DecimalOne()))
}

// GetRedemptionRequestForDate prepares a list of redemptions requests that are valid for the given date. If the current
// redemption date is the last one then all share holders are redeeming their full share.
func (vs *VaultState) GetRedemptionRequestForDate(now time.Time) []*RedeemRequest {
	// if this is the last redemption date redeem for all parties and set the state of the vault to stopping
	if vs.nextRedemptionDateIndex == len(vs.vault.RedemptionDates)-1 {
		vs.redeemQueue = []*RedeemRequest{}
		for party := range vs.shareHolders {
			vs.redeemQueue = append(vs.redeemQueue, &RedeemRequest{
				Party:     party,
				Date:      now,
				Amount:    num.UintZero(),
				Remaining: num.UintZero(),
				Status:    types.RedeemStatusPending,
			})
		}
	}

	// collect the redeem request for this date from the queue
	redeemRequests := []*RedeemRequest{}
	for _, rr := range vs.redeemQueue {
		if !rr.Date.After(now) {
			redeemRequests = append(redeemRequests, rr)
		} else {
			break
		}
	}
	sort.Slice(redeemRequests, func(i, j int) bool {
		return redeemRequests[i].Party < redeemRequests[j].Party
	})
	return redeemRequests
}

// PrepareRedemptions takes the share holding map and the redemption requests for a given redemption date, and calculates given the
// redemption date's type the required redemption per party and the list of late redemptions.
// assumptions:
// 1. party may have more than one redeem request per redemption day - their redeem is capped by the min(share, request)
// 2. if this is the last redemption date for the vault the redemption requests will have an amount of zero.
func PrepareRedemptions(shareHolders map[string]num.Decimal, redeemRequests []*RedeemRequest, vaultBalance, vaultLiquidBalance *num.Uint, redemptionType types.RedemptionType, maxFraction num.Decimal) (map[string]*num.Uint, []*RedeemRequest) {
	partyToRedeemed := map[string]*num.Uint{}
	lateRedemptions := []*RedeemRequest{}

	availableAmount, _ := num.UintFromDecimal(maxFraction.Mul(vaultBalance.ToDecimal()))
	actualLiquidAmount, _ := num.UintFromDecimal(maxFraction.Mul(vaultLiquidBalance.ToDecimal()))

	// if the vault is empty, we're done here
	if availableAmount.IsZero() || len(shareHolders) == 0 {
		for _, rr := range redeemRequests {
			rr.Status = types.RedeemStatusCompleted
		}
		return partyToRedeemed, lateRedemptions
	}

	for _, rr := range redeemRequests {
		partyShareOfLiquid, _ := num.UintFromDecimal(shareHolders[rr.Party].Mul(actualLiquidAmount.ToDecimal()))
		partyShareOfTotal, _ := num.UintFromDecimal(shareHolders[rr.Party].Mul(availableAmount.ToDecimal()))
		alreadyRedeemedThisDate := num.UintZero()
		if amt, ok := partyToRedeemed[rr.Party]; ok {
			alreadyRedeemedThisDate = amt
		} else {
			partyToRedeemed[rr.Party] = num.UintZero()
		}
		var redeem *num.Uint
		if rr.Amount.IsZero() {
			if !alreadyRedeemedThisDate.IsZero() {
				rr.Status = types.RedeemStatusCompleted
				rr.Remaining = num.UintZero()
				continue
			}
			redeem = num.Min(num.UintZero().Sub(partyShareOfLiquid, alreadyRedeemedThisDate), partyShareOfTotal)
		} else if redemptionType == types.RedemptionTypeFreeCashOnly {
			if alreadyRedeemedThisDate.EQ(partyShareOfLiquid) {
				rr.Status = types.RedeemStatusCompleted
				rr.Remaining = num.UintZero()
				continue
			}
			redeem = num.Min(num.UintZero().Sub(partyShareOfLiquid, alreadyRedeemedThisDate), rr.Amount)
		} else {
			redeem = num.Min(num.UintZero().Sub(partyShareOfLiquid, alreadyRedeemedThisDate), rr.Amount)
		}
		partyToRedeemed[rr.Party] = alreadyRedeemedThisDate.AddSum(redeem)
		if redemptionType == types.RedemptionTypeFreeCashOnly {
			rr.Status = types.RedeemStatusCompleted
			rr.Remaining = num.UintZero()
			continue
		}
		if !rr.Amount.IsZero() {
			if rr.Amount.EQ(redeem) {
				rr.Status = types.RedeemStatusCompleted
				rr.Remaining = num.UintZero()
			} else {
				rr.Status = types.RedeemStatusLate
				maxPartyShare, _ := num.UintFromDecimal(shareHolders[rr.Party].Mul(vaultBalance.ToDecimal()))
				rr.Remaining = num.UintZero().Sub(num.Min(rr.Amount, maxPartyShare), redeem)
				lateRedemptions = append(lateRedemptions, rr)
			}
		} else {
			if redeem.EQ(partyShareOfTotal) {
				rr.Status = types.RedeemStatusCompleted
				rr.Remaining = num.UintZero()
			} else {
				rr.Status = types.RedeemStatusLate
				lateRedemptions = append(lateRedemptions, rr)
			}
		}
	}
	return partyToRedeemed, lateRedemptions
}

func (vs *VaultState) GetVaultStatus() types.VaultStatus {
	return vs.status
}
