// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"context"
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type ChainStore interface {
	Get(context.Context) (entities.Chain, error)
	Set(context.Context, entities.Chain) error
}

type Chain struct {
	store ChainStore
	chain *entities.Chain
	mu    sync.Mutex
}

func NewChain(store ChainStore) *Chain {
	return &Chain{
		store: store,
	}
}

/*
GetChainID returns the current chain ID stored in the database (if one is set).

	If one is not set, return empty string and no error. If an error occurs,
	return empty string and an that error.

	It caches the result of calling to the store, so that once we have successfully
	retrieved a chain ID, we don't ask again.
*/
func (c *Chain) GetChainID() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.chain != nil {
		return c.chain.ID, nil
	}

	ctx := context.Background()
	chain, err := c.store.Get(ctx)
	if errors.Is(err, entities.ErrNotFound) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	c.chain = &chain
	return chain.ID, nil
}

func (c *Chain) SetChainID(chainID string) error {
	// Don't bother caching when we set, otherwise the code to fetch from the DB will never
	// be exercised until we start restoring mid-chain.
	ctx := context.Background()
	return c.store.Set(ctx, entities.Chain{ID: chainID})
}
