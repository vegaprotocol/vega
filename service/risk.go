package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type MarginLevelsStore interface {
	Add(marginLevel entities.MarginLevels) error
	Flush(ctx context.Context) ([]entities.MarginLevels, error)
	GetMarginLevelsByID(ctx context.Context, partyID, marketID string, pagination entities.OffsetPagination) ([]entities.MarginLevels, error)
	GetMarginLevelsByIDWithCursorPagination(ctx context.Context, partyID, marketID string, pagination entities.CursorPagination) ([]entities.MarginLevels, entities.PageInfo, error)
}

type AccountSource interface {
	GetByID(id int64) (entities.Account, error)
}

type Risk struct {
	mlStore       MarginLevelsStore
	accountSource AccountSource
	log           *logging.Logger
	observer      utils.Observer[entities.MarginLevels]
}

func NewRisk(mlStore MarginLevelsStore, accountSource AccountSource, log *logging.Logger) *Risk {
	return &Risk{
		mlStore:       mlStore,
		accountSource: accountSource,
		log:           log,
		observer:      utils.NewObserver[entities.MarginLevels]("margin_levels", log, 0, 0),
	}
}

func (r *Risk) Add(marginLevel entities.MarginLevels) error {
	return r.mlStore.Add(marginLevel)
}

func (r *Risk) Flush(ctx context.Context) error {
	flushed, err := r.mlStore.Flush(ctx)
	if err != nil {
		return err
	}
	r.observer.Notify(flushed)
	return nil
}

func (r *Risk) GetMarginLevelsByID(ctx context.Context, partyID, marketID string, pagination entities.OffsetPagination) ([]entities.MarginLevels, error) {
	return r.mlStore.GetMarginLevelsByID(ctx, partyID, marketID, pagination)
}

func (r *Risk) GetMarginLevelsByIDWithCursorPagination(ctx context.Context, partyID, marketID string, pagination entities.CursorPagination) ([]entities.MarginLevels, entities.PageInfo, error) {
	return r.mlStore.GetMarginLevelsByIDWithCursorPagination(ctx, partyID, marketID, pagination)
}

func (r *Risk) ObserveMarginLevels(
	ctx context.Context, retries int, partyID, marketID string,
) (accountCh <-chan []entities.MarginLevels, ref uint64) {
	ch, ref := r.observer.Observe(ctx, retries,
		func(ml entities.MarginLevels) bool {
			acc, err := r.accountSource.GetByID(ml.AccountID)
			if err != nil {
				return false
			}
			return (len(marketID) == 0 || marketID == acc.MarketID.String()) &&
				(len(partyID) == 0 || partyID == acc.PartyID.String())
		})
	return ch, ref
}
