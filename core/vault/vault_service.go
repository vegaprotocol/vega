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
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/vault TimeService,Collateral

type Collateral interface {
	CreateVaultAccount(ctx context.Context, vaultPartyID, asset string) (string, error)
	CloseVaultAccount(ctx context.Context, vaultPartyID string) error
	GetVaultBalance(vaultKey, asset string) (*num.Uint, error)
	GetVaultLiquidBalance(vaultKey, asset string) (*num.Uint, error)
	DepositToVault(ctx context.Context, vaultKey, asset, party string, amount *num.Uint) (*types.LedgerMovement, error)
	WithdrawFromVault(ctx context.Context, vaultKey, asset, party string, amount *num.Uint) (*types.LedgerMovement, error)
}

// Broker send events.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

type TimeService interface {
	GetTimeNow() time.Time
}

type VaultService struct {
	log                   *logging.Logger
	collateral            Collateral
	timeService           TimeService
	broker                Broker
	minNoticePeriodInDays int64
	vaultIdToVault        map[string]*VaultState
	sortedVaultIDs        []string
	lock                  sync.RWMutex
}

func NewVaultService(log *logging.Logger, collateral Collateral, timeService TimeService, broker Broker) *VaultService {
	return &VaultService{
		vaultIdToVault: map[string]*VaultState{},
		sortedVaultIDs: []string{},
		collateral:     collateral,
		timeService:    timeService,
		broker:         broker,
		log:            log,
	}
}

func (vs *VaultService) OnMinimumNoticePeriodChanged(ctx context.Context, minNoticePeriod *num.Uint) error {
	vs.minNoticePeriodInDays = int64(minNoticePeriod.Uint64())
	return nil
}

// GetVaultShares returns a copy of the share holding map for the vault.
func (vs *VaultService) GetVaultShares(vaultID string) map[string]num.Decimal {
	vs.lock.RLock()
	defer vs.lock.RUnlock()
	vault, ok := vs.vaultIdToVault[vaultID]
	if !ok {
		return map[string]num.Decimal{}
	}
	return vault.GetVaultShares()
}

// GetVaultOwner returns a pointer to the public key of the owner of the given vault or nil if the vault does not exist.
func (vs *VaultService) GetVaultOwner(vaultID string) *string {
	vs.lock.RLock()
	defer vs.lock.RUnlock()
	vault, ok := vs.vaultIdToVault[vaultID]
	if !ok {
		return nil
	}
	return &vault.vault.Owner
}

// CreateVault creates a new vault from the given configuration. Error is returned if the vault could not be created.
func (vs *VaultService) CreateVault(ctx context.Context, vault *types.Vault) error {
	vs.lock.Lock()
	defer vs.lock.Unlock()
	if _, ok := vs.vaultIdToVault[vault.ID]; ok {
		return fmt.Errorf("vault id already exists")
	}

	for _, rd := range vault.RedemptionDates {
		if rd.RedemptionDate.Before(time.Now()) {
			return fmt.Errorf("redemption dates are not allowed to be in the past")
		}
	}

	_, err := vs.collateral.CreateVaultAccount(ctx, vault.ID, vault.Asset)
	if err != nil {
		return err
	}

	vs.vaultIdToVault[vault.ID] = NewVaultState(vs.log, vault, vs.collateral, vs.timeService.GetTimeNow(), vs.broker)
	vs.sortedVaultIDs = append(vs.sortedVaultIDs, vault.ID)
	sort.Strings(vs.sortedVaultIDs)
	return nil
}

// UpdateVault updates an existing vault configuration. If the update fails an error is returned.
func (vs *VaultService) UpdateVault(ctx context.Context, vault *types.Vault) error {
	vs.lock.Lock()
	defer vs.lock.Unlock()
	if _, ok := vs.vaultIdToVault[vault.ID]; !ok {
		return fmt.Errorf("vault not found")
	}

	existing := vs.vaultIdToVault[vault.ID]
	if vault.Owner != existing.vault.Owner {
		return fmt.Errorf("only vault owner can update the vault state")
	}
	return vs.vaultIdToVault[vault.ID].UpdateVault(vault, vs.timeService.GetTimeNow(), vs.minNoticePeriodInDays)
}

// ChangeVaultOwnership changes the public key of the owner of the vault. If the update fails an error is returned.
func (vs *VaultService) ChangeVaultOwnership(ctx context.Context, vaultID, owner, newOwner string) error {
	vs.lock.Lock()
	defer vs.lock.Unlock()
	if _, ok := vs.vaultIdToVault[vaultID]; !ok {
		return fmt.Errorf("vault not found")
	}
	return vs.vaultIdToVault[vaultID].ChangeOwner(ctx, owner, newOwner)
}

// DepositToVault moves funds from the party general account to the vault general account.
func (vs *VaultService) DepositToVault(ctx context.Context, party, vaultKey string, amount *num.Uint) error {
	if _, ok := vs.vaultIdToVault[vaultKey]; !ok {
		return fmt.Errorf("vault does not exist")
	}
	vault := vs.vaultIdToVault[vaultKey]
	return vault.DepositToVault(ctx, party, amount)
}

// WithdrawFromVault generates a pending redeem request and adds it to the queue.
func (vs *VaultService) WithdrawFromVault(ctx context.Context, party, vaultKey string, amount *num.Uint) error {
	if _, ok := vs.vaultIdToVault[vaultKey]; !ok {
		return fmt.Errorf("vault does not exist")
	}
	vault := vs.vaultIdToVault[vaultKey]
	return vault.WithdrawFromVault(ctx, party, amount, vs.timeService.GetTimeNow())
}

// OnTick is called for every new block. We do the following for each vault:
// 1. process late redemption - if there are available funds in the general account use them for outstanding redemptions.
// 2. process fees if it's time.
// 3. process outstanding withdrawals if it's a redemption date for the vault.
func (vs *VaultService) OnTick(ctx context.Context, now time.Time) {
	for _, vaultID := range vs.sortedVaultIDs {
		vault := vs.vaultIdToVault[vaultID]
		vault.ProcessLateRedemptions(ctx)
		if !vault.nextFeeCalc.Before(now) {
			vault.ProcessFees(now)
		}
		vault.ProcessWithdrawals(ctx, now)
		if vault.status == types.VaultStatusStopped {
			delete(vs.vaultIdToVault, vaultID)
		}
	}
}
