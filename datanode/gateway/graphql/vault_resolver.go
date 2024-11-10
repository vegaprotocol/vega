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

package gql

import (
	"context"

	"code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type vaultResolver VegaResolverRoot

// Asset implements VaultResolver.
func (v *vaultResolver) Asset(ctx context.Context, obj *vega.Vault) (*vega.Asset, error) {
	return v.r.getAssetByID(ctx, obj.Asset)
}

// RedemptionDates implements VaultResolver.
func (v *vaultResolver) RedemptionDates(ctx context.Context, obj *vega.Vault) ([]*RedemptionDate, error) {
	rd := make([]*RedemptionDate, 0, len(obj.RedemptionDates))
	for _, d := range obj.RedemptionDates {
		rd = append(rd, &RedemptionDate{
			MaxFraction:    d.MaxFraction,
			RedemptionType: RedemptionType(d.RedemptionType),
			RedemptionDate: d.RedemptionDate,
		})
	}
	return rd, nil
}

type vaultStateResolver VegaResolverRoot

// Asset implements RedemptionRequestResolver.
func (v *vaultStateResolver) Asset(ctx context.Context, obj *v1.RedemptionRequest) (*vega.Asset, error) {
	return v.r.getAssetByID(ctx, obj.Asset)
}

// EligibilityDate implements RedemptionRequestResolver.
func (v *vaultStateResolver) EligibilityDate(ctx context.Context, obj *v1.RedemptionRequest) (*int64, error) {
	return &obj.Date, nil
}

// LastUpdated implements RedemptionRequestResolver.
func (v *vaultStateResolver) LastUpdated(ctx context.Context, obj *v1.RedemptionRequest) (*int64, error) {
	return &obj.LastUpdate, nil
}

// Status implements RedemptionRequestResolver.
func (v *vaultStateResolver) Status(ctx context.Context, obj *v1.RedemptionRequest) (RedeemStatus, error) {
	if obj.Status == vega.RedeemStatus_REDEEM_STATUS_PENDING {
		return RedeemStatusPending, nil
	}
	if obj.Status == vega.RedeemStatus_REDEEM_STATUS_COMPLETED {
		return RedeemStatusCompleted, nil
	}
	if obj.Status == vega.RedeemStatus_REDEEM_STATUS_LATE {
		return RedeemStatusLate, nil
	}
	return RedeemStatusUnspecified, nil
}

// PartyShares implements VaultStateResolver.
func (v *vaultStateResolver) PartyShares(ctx context.Context, obj *v1.VaultState) ([]*PartyVaultShare, error) {
	pvs := make([]*PartyVaultShare, 0, len(obj.PartyShares))
	for _, ps := range obj.PartyShares {
		pvs = append(pvs, &PartyVaultShare{
			PartyID: ps.Party,
			Share:   ps.Share,
		})
	}
	return pvs, nil
}

// VaultStatus implements VaultStateResolver.
func (v *vaultStateResolver) VaultStatus(ctx context.Context, obj *v1.VaultState) (VaultStatus, error) {
	if obj.Status == vega.VaultStatus_VAULT_STATUS_ACTIVE {
		return VaultStatusActive, nil
	}
	if obj.Status == vega.VaultStatus_VAULT_STATUS_STOPPING {
		return VaultStatusStopping, nil
	}
	if obj.Status == vega.VaultStatus_VAULT_STATUS_STOPPED {
		return VaultStatusStopped, nil
	}
	return VaultStatusUnspecified, nil
}
