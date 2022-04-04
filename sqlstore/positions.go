package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

var ErrPositionNotFound = errors.New("party not found")

type Positions struct {
	*SQLStore
	cache     map[entities.MarketID]map[entities.PartyID]entities.Position
	cacheLock sync.Mutex
}

func NewPositions(sqlStore *SQLStore) *Positions {
	a := &Positions{
		SQLStore:  sqlStore,
		cache:     map[entities.MarketID]map[entities.PartyID]entities.Position{},
		cacheLock: sync.Mutex{},
	}
	return a
}

func (ps *Positions) Add(ctx context.Context, p entities.Position) error {
	ps.cacheLock.Lock()
	defer ps.cacheLock.Unlock()

	_, err := ps.pool.Exec(ctx,
		`INSERT INTO positions(market_id, party_id, open_volume, realised_pnl, unrealised_pnl, average_entry_price, loss, adjustment, vega_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT(market_id, party_id, vega_time)
		 DO UPDATE SET
		    open_volume=EXCLUDED.open_volume,
		    realised_pnl=EXCLUDED.realised_pnl,
		    unrealised_pnl=EXCLUDED.unrealised_pnl,
		    average_entry_price=EXCLUDED.average_entry_price,
			loss=EXCLUDED.loss,
			adjustment=EXCLUDED.adjustment
		 `,
		p.MarketID,
		p.PartyID,
		p.OpenVolume,
		p.RealisedPnl,
		p.UnrealisedPnl,
		p.AverageEntryPrice,
		p.Loss,
		p.Adjustment,
		p.VegaTime)

	ps.updateCache(p)
	return err
}

func (ps *Positions) GetByMarketAndParty(ctx context.Context,
	marketID entities.MarketID,
	partyID entities.PartyID,
) (entities.Position, error) {
	ps.cacheLock.Lock()
	defer ps.cacheLock.Unlock()

	position, found := ps.checkCache(marketID, partyID)
	if found {
		return position, nil
	}

	err := pgxscan.Get(ctx, ps.pool, &position,
		`SELECT * FROM positions_current WHERE market_id=$1 AND party_id=$2`,
		marketID, partyID)

	if err == nil {
		ps.updateCache(position)
	}

	if pgxscan.NotFound(err) {
		return position, fmt.Errorf("'%v/%v': %w", marketID, partyID, ErrPositionNotFound)
	}

	return position, err
}

func (ps *Positions) GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error) {
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.pool, &positions,
		`SELECT * FROM positions_current WHERE market_id=$1`,
		marketID)
	return positions, err
}

func (ps *Positions) GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error) {
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.pool, &positions,
		`SELECT * FROM positions_current WHERE party_id=$1`,
		partyID)
	return positions, err
}

func (ps *Positions) GetAll(ctx context.Context) ([]entities.Position, error) {
	positions := []entities.Position{}
	err := pgxscan.Select(ctx, ps.pool, &positions,
		`SELECT * FROM positions_current`)
	return positions, err
}

func (ps *Positions) updateCache(p entities.Position) {
	if _, ok := ps.cache[p.MarketID]; !ok {
		ps.cache[p.MarketID] = map[entities.PartyID]entities.Position{}
	}

	ps.cache[p.MarketID][p.PartyID] = p
}

func (ps *Positions) checkCache(marketID entities.MarketID, partyID entities.PartyID) (entities.Position, bool) {
	if _, ok := ps.cache[marketID]; !ok {
		return entities.Position{}, false
	}

	pos, ok := ps.cache[marketID][partyID]
	if !ok {
		return entities.Position{}, false
	}
	return pos, true
}
