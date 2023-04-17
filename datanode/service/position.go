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

	lru "github.com/hashicorp/golang-lru"
)

type PositionStore interface {
	Flush(ctx context.Context) ([]entities.Position, error)
	Add(ctx context.Context, p entities.Position) error
	GetByMarketAndParty(ctx context.Context, marketID string, partyID string) (entities.Position, error)
	GetByMarketAndParties(ctx context.Context, marketIDRaw string, partyIDsRaw []string) ([]entities.Position, error)
	GetByMarket(ctx context.Context, marketID string) ([]entities.Position, error)
	GetByParty(ctx context.Context, partyID string) ([]entities.Position, error)
	GetByPartyConnection(ctx context.Context, partyID []string, marketID []string, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Position, error)
	GetAll(ctx context.Context) ([]entities.Position, error)
}

type positionCacheKey struct {
	MarketID entities.MarketID
	PartyID  entities.PartyID
}
type Position struct {
	store    PositionStore
	observer utils.Observer[entities.Position]
	cache    *lru.Cache
}

func NewPosition(store PositionStore, log *logging.Logger) *Position {
	cache, err := lru.New(10000)
	if err != nil {
		panic(err)
	}
	return &Position{
		store:    store,
		observer: utils.NewObserver[entities.Position]("positions", log, 0, 0),
		cache:    cache,
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
	key := positionCacheKey{pos.MarketID, pos.PartyID}
	p.cache.Add(key, pos)
	return p.store.Add(ctx, pos)
}

func (p *Position) GetByMarketAndParties(ctx context.Context, marketID string, partyIDs []string) ([]entities.Position, error) {
	missedParties := make([]string, 0, len(partyIDs))
	ret := make([]entities.Position, 0, len(partyIDs))
	key := positionCacheKey{
		MarketID: entities.MarketID(marketID),
	}
	for _, partyID := range partyIDs {
		key.PartyID = entities.PartyID(partyID)
		if v, ok := p.cache.Get(key); ok {
			switch val := v.(type) {
			case entities.Position:
				ret = append(ret, val)
			default:
				// this includes errors from cache, ignore them and try again?
				missedParties = append(missedParties, partyID)
			}
		} else {
			missedParties = append(missedParties, partyID)
		}
	}
	// everything was cached, we're done
	if len(missedParties) == 0 {
		return ret, nil
	}
	storePos, err := p.store.GetByMarketAndParties(ctx, marketID, missedParties)
	// append the positions from store to those from cache
	ret = append(ret, storePos...)
	if err == nil {
		// we had cache misses, and got them from store, so add them to cache
		for _, sp := range storePos {
			key.PartyID = sp.PartyID
			p.cache.Add(key, sp)
		}
	}
	return ret, err
}

func (p *Position) GetByMarketAndParty(ctx context.Context, marketID string, partyID string) (entities.Position, error) {
	key := positionCacheKey{entities.MarketID(marketID), entities.PartyID(partyID)}
	value, ok := p.cache.Get(key)
	if ok {
		// make sure the value in cache is a position entity, ignore errors
		if v, ok := value.(entities.Position); ok {
			return v, nil
		}
	}
	// either cache miss, or an error was cached, either way fall back to store and update cache
	pos, err := p.store.GetByMarketAndParty(
		ctx, marketID, partyID)
	// let's not cache errors here
	if err == nil {
		p.cache.Add(key, pos)
	}

	return pos, err
}

func (p *Position) GetByMarket(ctx context.Context, marketID string) ([]entities.Position, error) {
	return p.store.GetByMarket(ctx, marketID)
}

func (p *Position) GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error) {
	return p.store.GetByParty(ctx, partyID.String())
}

func (p *Position) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Position, error) {
	return p.store.GetByTxHash(ctx, txHash)
}

func (p *Position) GetByPartyConnection(ctx context.Context, partyIDs []entities.PartyID, marketIDs []entities.MarketID, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error) {
	ps := make([]string, len(partyIDs))
	for i, p := range partyIDs {
		ps[i] = p.String()
	}

	ms := make([]string, len(marketIDs))
	for i, m := range marketIDs {
		ms[i] = m.String()
	}
	return p.store.GetByPartyConnection(ctx, ps, ms, pagination)
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
