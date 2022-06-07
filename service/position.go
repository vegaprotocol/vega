package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type positionStore interface {
	Flush(ctx context.Context) ([]entities.Position, error)
	Add(ctx context.Context, p entities.Position) error
	GetByMarketAndParty(ctx context.Context, marketID entities.MarketID, partyID entities.PartyID) (entities.Position, error)
	GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error)
	GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error)
	GetAll(ctx context.Context) ([]entities.Position, error)
}

type Position struct {
	log      *logging.Logger
	store    positionStore
	observer utils.Observer[entities.Position]
}

func NewPosition(store positionStore, log *logging.Logger) *Position {
	return &Position{
		store:    store,
		log:      log,
		observer: utils.NewObserver[entities.Position]("positions", log, 0, 0),
	}
}

func (p *Position) Flush(ctx context.Context) error {
	flushed, err := p.store.Flush(ctx)
	if err != nil {
		return err
	}
	p.observer.Notify(flushed)
	return nil
}

func (p *Position) Add(ctx context.Context, pos entities.Position) error {
	return p.store.Add(ctx, pos)
}

func (p *Position) GetByMarketAndParty(ctx context.Context, marketID entities.MarketID, partyID entities.PartyID) (entities.Position, error) {
	return p.store.GetByMarketAndParty(ctx, marketID, partyID)
}

func (p *Position) GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error) {
	return p.store.GetByMarket(ctx, marketID)
}

func (p *Position) GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error) {
	return p.store.GetByParty(ctx, partyID)
}

func (p *Position) GetAll(ctx context.Context) ([]entities.Position, error) {
	return p.store.GetAll(ctx)
}

func (p *Position) Observe(ctx context.Context, retries int, partyID, marketID string) (<-chan []entities.Position, uint64) {
	ch, ref := p.observer.Observe(ctx,
		retries,
		func(pos entities.Position) bool {
			return (len(marketID) == 0 || marketID == pos.MarketID.String()) &&
				(len(partyID) == 0 || partyID == pos.PartyID.String())
		})
	return ch, ref
}
