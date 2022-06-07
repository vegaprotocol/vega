package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type orderStore interface {
	Flush(ctx context.Context) ([]entities.Order, error)
	Add(o entities.Order) error
	GetAll(ctx context.Context) ([]entities.Order, error)
	GetByOrderID(ctx context.Context, orderIdStr string, version *int32) (entities.Order, error)
	GetByMarket(ctx context.Context, marketIdStr string, p entities.OffsetPagination) ([]entities.Order, error)
	GetByParty(ctx context.Context, partyIdStr string, p entities.OffsetPagination) ([]entities.Order, error)
	GetByReference(ctx context.Context, reference string, p entities.OffsetPagination) ([]entities.Order, error)
	GetAllVersionsByOrderID(ctx context.Context, id string, p entities.OffsetPagination) ([]entities.Order, error)
	GetLiveOrders(ctx context.Context) ([]entities.Order, error)
	GetByMarketPaged(ctx context.Context, marketIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error)
	GetByPartyPaged(ctx context.Context, partyIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error)
	GetOrderVersionsByIDPaged(ctx context.Context, orderIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error)
	GetByPartyAndMarketPaged(ctx context.Context, partyIDStr, marketIDStr string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error)
}

type Order struct {
	store    orderStore
	log      *logging.Logger
	observer utils.Observer[entities.Order]
}

func NewOrder(store orderStore, log *logging.Logger) *Order {
	return &Order{
		store:    store,
		log:      log,
		observer: utils.NewObserver[entities.Order]("order", log, 0, 0),
	}
}

func (o *Order) ObserveOrders(ctx context.Context, retries int, market *string, party *string) (<-chan []entities.Order, uint64) {
	ch, ref := o.observer.Observe(ctx,
		retries,
		func(o entities.Order) bool {
			marketOk := market == nil || o.MarketID.String() == *market
			partyOk := party == nil || o.PartyID.String() == *party
			return marketOk && partyOk
		})
	return ch, ref
}

func (o *Order) Flush(ctx context.Context) error {
	flushed, err := o.store.Flush(ctx)
	if err != nil {
		return err
	}
	o.observer.Notify(flushed)
	return nil
}

func (o *Order) Add(order entities.Order) error {
	return o.store.Add(order)
}

func (o *Order) GetAll(ctx context.Context) ([]entities.Order, error) {
	return o.store.GetAll(ctx)
}

func (o *Order) GetByOrderID(ctx context.Context, orderID string, version *int32) (entities.Order, error) {
	return o.store.GetByOrderID(ctx, orderID, version)
}

func (o *Order) GetByMarket(ctx context.Context, marketID string, p entities.OffsetPagination) ([]entities.Order, error) {
	return o.store.GetByMarket(ctx, marketID, p)
}

func (o *Order) GetByParty(ctx context.Context, partyID string, p entities.OffsetPagination) ([]entities.Order, error) {
	return o.store.GetByParty(ctx, partyID, p)
}

func (o *Order) GetByReference(ctx context.Context, reference string, p entities.OffsetPagination) ([]entities.Order, error) {
	return o.store.GetByReference(ctx, reference, p)
}

func (o *Order) GetAllVersionsByOrderID(ctx context.Context, id string, p entities.OffsetPagination) ([]entities.Order, error) {
	return o.store.GetAllVersionsByOrderID(ctx, id, p)
}

func (o *Order) GetLiveOrders(ctx context.Context) ([]entities.Order, error) {
	return o.store.GetLiveOrders(ctx)
}

func (o *Order) GetByMarketPaged(ctx context.Context, marketID string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	return o.store.GetByMarketPaged(ctx, marketID, p)
}

func (o *Order) GetByPartyPaged(ctx context.Context, partyID string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	return o.store.GetByPartyPaged(ctx, partyID, p)
}

func (o *Order) GetOrderVersionsByIDPaged(ctx context.Context, orderID string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	return o.store.GetOrderVersionsByIDPaged(ctx, orderID, p)
}

func (o *Order) GetByPartyAndMarketPaged(ctx context.Context, partyID, marketID string, p entities.Pagination) ([]entities.Order, entities.PageInfo, error) {
	return o.store.GetByPartyAndMarketPaged(ctx, partyID, marketID, p)
}
