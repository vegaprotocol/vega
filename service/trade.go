package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type tradeStore interface {
	Flush(ctx context.Context) ([]*entities.Trade, error)
	Add(t *entities.Trade) error
	GetByMarket(ctx context.Context, market string, p entities.OffsetPagination) ([]entities.Trade, error)
	GetByMarketWithCursor(ctx context.Context, market string, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error)
	GetByParty(ctx context.Context, party string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error)
	GetByPartyWithCursor(ctx context.Context, party string, market *string, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error)
	GetByOrderID(ctx context.Context, order string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error)
	GetByOrderIDWithCursor(ctx context.Context, order string, market *string, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error)
}

type Trade struct {
	store    tradeStore
	log      *logging.Logger
	observer utils.Observer[*entities.Trade]
}

func NewTrade(store tradeStore, log *logging.Logger) *Trade {
	return &Trade{
		store:    store,
		log:      log,
		observer: utils.NewObserver[*entities.Trade]("trade", log, 0, 0),
	}
}

func (t *Trade) Flush(ctx context.Context) error {
	flushed, err := t.store.Flush(ctx)
	if err != nil {
		return err
	}
	t.observer.Notify(flushed)
	return nil
}

func (t *Trade) Add(trade *entities.Trade) error {
	return t.store.Add(trade)
}

func (t *Trade) GetByMarket(ctx context.Context, market string, p entities.OffsetPagination) ([]entities.Trade, error) {
	return t.store.GetByMarket(ctx, market, p)
}

func (t *Trade) GetByMarketWithCursor(ctx context.Context, market string, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error) {
	return t.store.GetByMarketWithCursor(ctx, market, pagination)
}

func (t *Trade) GetByParty(ctx context.Context, party string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error) {
	return t.store.GetByParty(ctx, party, market, pagination)
}

func (t *Trade) GetByPartyWithCursor(ctx context.Context, party string, market *string, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error) {
	return t.store.GetByPartyWithCursor(ctx, party, market, pagination)
}

func (t *Trade) GetByOrderID(ctx context.Context, order string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error) {
	return t.store.GetByOrderID(ctx, order, market, pagination)
}

func (t *Trade) GetByOrderIDWithCursor(ctx context.Context, order string, market *string, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error) {
	return t.store.GetByOrderIDWithCursor(ctx, order, market, pagination)
}

func (t *Trade) Observe(ctx context.Context, retries int, marketID *string, partyID *string) (<-chan []*entities.Trade, uint64) {
	ch, ref := t.observer.Observe(ctx,
		retries,
		func(trade *entities.Trade) bool {
			return (marketID == nil || *marketID == trade.MarketID.String()) &&
				(partyID == nil || *partyID == trade.Buyer.String() || *partyID == trade.Seller.String())
		})
	return ch, ref
}
