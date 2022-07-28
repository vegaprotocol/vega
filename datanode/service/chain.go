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
	"errors"
	"sync"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/logging"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/service ChainStore
type ChainStore interface {
	Get(context.Context) (entities.Chain, error)
	Set(context.Context, entities.Chain) error
}

type Chain struct {
	store ChainStore
	chain *entities.Chain
	log   *logging.Logger
	mu    sync.Mutex
}

func NewChain(store ChainStore, log *logging.Logger) *Chain {
	return &Chain{
		store: store,
		log:   log,
	}
}

/* GetChainID returns the current chain ID stored in the database (if one is set).
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
	if errors.Is(err, entities.ErrChainNotFound) {
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
