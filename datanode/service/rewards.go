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
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
)

type rewardStore interface {
	Add(ctx context.Context, r entities.Reward) error
	GetAll(ctx context.Context) ([]entities.Reward, error)
	GetByCursor(ctx context.Context, partyID *string, assetID *string, fromEpoch, toEpoch *uint64, p entities.CursorPagination) ([]entities.Reward, entities.PageInfo, error)
	GetSummaries(ctx context.Context, partyID *string, assetID *string) ([]entities.RewardSummary, error)
	GetEpochSummaries(ctx context.Context, filter entities.RewardSummaryFilter, p entities.CursorPagination) ([]entities.EpochRewardSummary, entities.PageInfo, error)
}

type Reward struct {
	log      *logging.Logger
	store    rewardStore
	observer utils.Observer[entities.Reward]
}

func NewReward(store rewardStore, log *logging.Logger) *Reward {
	return &Reward{
		store:    store,
		log:      log,
		observer: utils.NewObserver[entities.Reward]("reward", log, 0, 0),
	}
}

func (r *Reward) Add(ctx context.Context, reward entities.Reward) error {
	err := r.store.Add(ctx, reward)
	if err != nil {
		return err
	}
	r.observer.Notify([]entities.Reward{reward})
	return nil
}

func (r *Reward) GetAll(ctx context.Context) ([]entities.Reward, error) {
	return r.store.GetAll(ctx)
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

func (r *Reward) Observe(ctx context.Context, retries int, assetID, partyID string) (rewardCh <-chan []entities.Reward, ref uint64) {
	ch, ref := r.observer.Observe(ctx,
		retries,
		func(reward entities.Reward) bool {
			return (len(assetID) == 0 || assetID == reward.AssetID.String()) &&
				(len(partyID) == 0 || partyID == reward.PartyID.String())
		})
	return ch, ref
}

func (r *Reward) GetRewardSubscribersCount() int32 {
	return r.observer.GetSubscribersCount()
}
