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
)

type VaultStore interface {
	Add(ctx context.Context, vaultState *entities.VaultState) error
	ListVaultsWithCursor(ctx context.Context, vaultIDs []string, assetIDs []string, liveOnly bool, pagination entities.CursorPagination) ([]entities.VaultState, entities.PageInfo, error)
}

type Vault struct {
	store VaultStore
}

func NewVault(store VaultStore, log *logging.Logger) *Vault {
	return &Vault{
		store: store,
	}
}

func (v *Vault) Add(ctx context.Context, vaultState entities.VaultState) error {
	err := v.store.Add(ctx, &vaultState)
	if err != nil {
		return err
	}
	return nil
}

func (v *Vault) ListVaultsWithCursor(ctx context.Context,
	vaultIDs []string,
	assetIDs []string,
	liveOnly bool,
	pagination entities.CursorPagination,
) ([]entities.VaultState, entities.PageInfo, error) {
	return v.store.ListVaultsWithCursor(ctx, vaultIDs, assetIDs, liveOnly, pagination)
}
