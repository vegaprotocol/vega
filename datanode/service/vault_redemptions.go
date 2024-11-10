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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type VaultRedemptionsStore interface {
	Add(ctx context.Context, redemptionRequest *entities.RedemptionRequest) error
	ListRedemptionRequestsWithCursor(ctx context.Context, vaultIDs, partyIDs, assetIDs []string, status []vega.RedeemStatus, pagination entities.CursorPagination) ([]entities.RedemptionRequest, entities.PageInfo, error)
}

type VaultRedemptions struct {
	store VaultRedemptionsStore
}

func NewVaultRedemptions(store VaultRedemptionsStore, log *logging.Logger) *VaultRedemptions {
	return &VaultRedemptions{
		store: store,
	}
}

func (vr *VaultRedemptions) Add(ctx context.Context, request entities.RedemptionRequest) error {
	err := vr.store.Add(ctx, &request)
	if err != nil {
		return err
	}
	return nil
}

func (vr *VaultRedemptions) ListRedemptionRequestsWithCursor(ctx context.Context, vaultIDs, partyIDs, assetIDs []string, statuses []vega.RedeemStatus, pagination entities.CursorPagination) ([]entities.RedemptionRequest, entities.PageInfo, error) {
	return vr.store.ListRedemptionRequestsWithCursor(ctx, vaultIDs, partyIDs, assetIDs, statuses, pagination)
}
