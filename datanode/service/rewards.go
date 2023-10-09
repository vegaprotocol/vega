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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

type rewardStore interface {
	Add(ctx context.Context, r entities.Reward) error
	GetAll(ctx context.Context) ([]entities.Reward, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Reward, error)
	GetByCursor(ctx context.Context, partyID *string, assetID *string, fromEpoch, toEpoch *uint64, p entities.CursorPagination) ([]entities.Reward, entities.PageInfo, error)
	GetSummaries(ctx context.Context, partyID *string, assetID *string) ([]entities.RewardSummary, error)
	GetEpochSummaries(ctx context.Context, filter entities.RewardSummaryFilter, p entities.CursorPagination) ([]entities.EpochRewardSummary, entities.PageInfo, error)
}

type Reward struct {
	store rewardStore
}

func NewReward(store rewardStore, log *logging.Logger) *Reward {
	return &Reward{
		store: store,
	}
}

func (r *Reward) Add(ctx context.Context, reward entities.Reward) error {
	err := r.store.Add(ctx, reward)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reward) GetAll(ctx context.Context) ([]entities.Reward, error) {
	return r.store.GetAll(ctx)
}

func (r *Reward) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Reward, error) {
	return r.store.GetByTxHash(ctx, txHash)
}

func (r *Reward) GetByCursor(ctx context.Context, partyID, assetID *string, fromEpoch, toEpoch *uint64, p entities.CursorPagination) ([]entities.Reward, entities.PageInfo, error) {
	return r.store.GetByCursor(ctx, partyID, assetID, fromEpoch, toEpoch, p)
}

func (r *Reward) GetSummaries(ctx context.Context, partyID *string, assetID *string) ([]entities.RewardSummary, error) {
	return r.store.GetSummaries(ctx, partyID, assetID)
}

func (r *Reward) GetEpochRewardSummaries(ctx context.Context, filter entities.RewardSummaryFilter, p entities.CursorPagination) ([]entities.EpochRewardSummary, entities.PageInfo, error) {
	return r.store.GetEpochSummaries(ctx, filter, p)
}
