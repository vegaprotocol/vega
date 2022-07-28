// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/utils"
	"code.vegaprotocol.io/data-node/logging"
	lru "github.com/hashicorp/golang-lru"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/position_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/service PositionStore
type PositionStore interface {
	Flush(ctx context.Context) ([]entities.Position, error)
	Add(ctx context.Context, p entities.Position) error
	GetByMarketAndParty(ctx context.Context, marketID entities.MarketID, partyID entities.PartyID) (entities.Position, error)
	GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error)
	GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error)
	GetByPartyConnection(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error)
	GetAll(ctx context.Context) ([]entities.Position, error)
}

type positionCacheKey struct {
	MarketID entities.MarketID
	PartyID  entities.PartyID
}
type Position struct {
	log      *logging.Logger
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
		log:      log,
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

func (p *Position) GetByMarketAndParty(ctx context.Context, marketID entities.MarketID, partyID entities.PartyID) (entities.Position, error) {
	key := positionCacheKey{marketID, partyID}
	value, ok := p.cache.Get(key)
	if !ok {
		pos, err := p.store.GetByMarketAndParty(ctx, marketID, partyID)
		if err == nil {
			p.cache.Add(key, pos)
		} else { // If store errors in the cache too
			p.cache.Add(key, err)
		}

		return pos, err
	}

	switch v := value.(type) {
	case entities.Position:
		return v, nil
	case error:
		return entities.Position{}, v
	default:
		return entities.Position{}, fmt.Errorf("unknown type in cache")
	}
}

func (p *Position) GetByMarket(ctx context.Context, marketID entities.MarketID) ([]entities.Position, error) {
	return p.store.GetByMarket(ctx, marketID)
}

func (p *Position) GetByParty(ctx context.Context, partyID entities.PartyID) ([]entities.Position, error) {
	return p.store.GetByParty(ctx, partyID)
}

func (p *Position) GetByPartyConnection(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID, pagination entities.CursorPagination) ([]entities.Position, entities.PageInfo, error) {
	return p.store.GetByPartyConnection(ctx, partyID, marketID, pagination)
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
