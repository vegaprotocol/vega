package service

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type MarketDataStore interface {
	Add(data *entities.MarketData) error
	Flush(ctx context.Context) ([]*entities.MarketData, error)
	GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error)
	GetMarketsData(ctx context.Context) ([]entities.MarketData, error)
	GetBetweenDatesByID(ctx context.Context, marketID string, start, end time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error)
	GetFromDateByID(ctx context.Context, marketID string, start time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error)
	GetToDateByID(ctx context.Context, marketID string, end time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error)
}

type MarketData struct {
	store    MarketDataStore
	log      *logging.Logger
	observer utils.Observer[*entities.MarketData]
}

func NewMarketData(store MarketDataStore, log *logging.Logger) *MarketData {
	return &MarketData{
		log:      log,
		store:    store,
		observer: utils.NewObserver[*entities.MarketData]("market_data", log, 0, 0),
	}
}

func (m *MarketData) Add(data *entities.MarketData) error {
	return m.store.Add(data)
}

func (m *MarketData) Flush(ctx context.Context) error {
	flushed, err := m.store.Flush(ctx)
	if err != nil {
		return err
	}
	m.observer.Notify(flushed)
	return nil
}

func (m *MarketData) GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error) {
	return m.store.GetMarketDataByID(ctx, marketID)
}

func (m *MarketData) GetMarketsData(ctx context.Context) ([]entities.MarketData, error) {
	return m.store.GetMarketsData(ctx)
}

func (m *MarketData) GetBetweenDatesByID(ctx context.Context, marketID string, start, end time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error) {
	return m.store.GetBetweenDatesByID(ctx, marketID, start, end, pagination)
}

func (m *MarketData) GetFromDateByID(ctx context.Context, marketID string, start time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error) {
	return m.store.GetFromDateByID(ctx, marketID, start, pagination)
}

func (m *MarketData) GetToDateByID(ctx context.Context, marketID string, end time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error) {
	return m.store.GetToDateByID(ctx, marketID, end, pagination)
}

func (m *MarketData) ObserveMarketData(
	ctx context.Context, retries int, marketID string,
) (<-chan []*entities.MarketData, uint64) {
	ch, ref := m.observer.Observe(ctx,
		retries,
		func(md *entities.MarketData) bool { return len(marketID) == 0 || marketID == md.Market.String() })
	return ch, ref
}
