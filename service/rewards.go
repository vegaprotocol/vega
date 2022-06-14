package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type rewardStore interface {
	Add(ctx context.Context, r entities.Reward) error
	GetAll(ctx context.Context) ([]entities.Reward, error)
	GetByOffset(ctx context.Context, partyID *string, assetID *string, p *entities.OffsetPagination) ([]entities.Reward, error)
	GetByCursor(ctx context.Context, partyID *string, assetID *string, p entities.CursorPagination) ([]entities.Reward, entities.PageInfo, error)
	GetSummaries(ctx context.Context, partyID *string, assetID *string) ([]entities.RewardSummary, error)
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

func (r *Reward) GetByOffset(ctx context.Context, partyID *string, assetID *string, p *entities.OffsetPagination) ([]entities.Reward, error) {
	return r.store.GetByOffset(ctx, partyID, assetID, p)
}

func (r *Reward) GetByCursor(ctx context.Context, partyID, assetID *string, p entities.CursorPagination) ([]entities.Reward, entities.PageInfo, error) {
	return r.store.GetByCursor(ctx, partyID, assetID, p)
}

func (r *Reward) GetSummaries(ctx context.Context, partyID *string, assetID *string) ([]entities.RewardSummary, error) {
	return r.store.GetSummaries(ctx, partyID, assetID)
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
