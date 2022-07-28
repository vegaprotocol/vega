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

package sqlstore

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type Assets struct {
	*ConnectionSource
	cache     map[string]entities.Asset
	cacheLock sync.Mutex
}

func NewAssets(connectionSource *ConnectionSource) *Assets {
	a := &Assets{
		ConnectionSource: connectionSource,
		cache:            make(map[string]entities.Asset),
	}
	return a
}

func (as *Assets) Add(ctx context.Context, a entities.Asset) error {
	defer metrics.StartSQLQuery("Assets", "Add")()
	_, err := as.Connection.Exec(ctx,
		`INSERT INTO assets(id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, vega_time, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
         ON CONFLICT (id, vega_time) DO UPDATE SET
            name = EXCLUDED.name,
            symbol = EXCLUDED.symbol,
            total_supply = EXCLUDED.total_supply,
            decimals = EXCLUDED.decimals,
            quantum = EXCLUDED.quantum,
            source = EXCLUDED.source,
            erc20_contract = EXCLUDED.erc20_contract,
            lifetime_limit = EXCLUDED.lifetime_limit,
            withdraw_threshold = EXCLUDED.withdraw_threshold,
            vega_time = EXCLUDED.vega_time,
            status = EXCLUDED.status
            ;`,
		a.ID,
		a.Name,
		a.Symbol,
		a.TotalSupply,
		a.Decimals,
		a.Quantum,
		a.Source,
		a.ERC20Contract,
		a.LifetimeLimit,
		a.WithdrawThreshold,
		a.VegaTime,
		a.Status,
	)
	if err != nil {
		return err
	}

	// delete cache
	as.cacheLock.Lock()
	defer as.cacheLock.Unlock()
	delete(as.cache, a.ID.String())

	return nil
}

func (as *Assets) GetByID(ctx context.Context, id string) (entities.Asset, error) {
	as.cacheLock.Lock()
	defer as.cacheLock.Unlock()

	if asset, ok := as.cache[id]; ok {
		return asset, nil
	}

	a := entities.Asset{}

	defer metrics.StartSQLQuery("Assets", "GetByID")()
	err := pgxscan.Get(ctx, as.Connection, &a,
		getAssetQuery()+` WHERE id=$1`,
		entities.NewAssetID(id))

	if err == nil {
		as.cache[id] = a
	}
	return a, err
}

func (as *Assets) GetAll(ctx context.Context) ([]entities.Asset, error) {
	assets := []entities.Asset{}
	defer metrics.StartSQLQuery("Assets", "GetAll")()
	err := pgxscan.Select(ctx, as.Connection, &assets, getAssetQuery())
	return assets, err
}

func (as *Assets) GetAllWithCursorPagination(ctx context.Context, pagination entities.CursorPagination) (
	[]entities.Asset, entities.PageInfo, error) {
	var assets []entities.Asset
	var pageInfo entities.PageInfo
	var args []interface{}

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("id", sorting, cmp, entities.NewAssetID(cursor)),
	}

	query := getAssetQuery()
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	defer metrics.StartSQLQuery("Assets", "GetAllWithCursorPagination")()

	if err := pgxscan.Select(ctx, as.Connection, &assets, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get assets: %w", err)
	}

	assets, pageInfo = entities.PageEntities[*v2.AssetEdge](assets, pagination)

	return assets, pageInfo, nil
}

func getAssetQuery() string {
	return `SELECT * FROM assets_current`
}
