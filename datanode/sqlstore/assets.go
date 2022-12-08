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

package sqlstore

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var assetOrdering = TableOrdering{
	ColumnOrdering{Name: "id", Sorting: ASC},
}

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
		`INSERT INTO assets(id, name, symbol, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, tx_hash, vega_time, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
         ON CONFLICT (id, vega_time) DO UPDATE SET
            name = EXCLUDED.name,
            symbol = EXCLUDED.symbol,
            decimals = EXCLUDED.decimals,
            quantum = EXCLUDED.quantum,
            source = EXCLUDED.source,
            erc20_contract = EXCLUDED.erc20_contract,
            lifetime_limit = EXCLUDED.lifetime_limit,
            withdraw_threshold = EXCLUDED.withdraw_threshold,
			tx_hash = EXCLUDED.tx_hash,
            vega_time = EXCLUDED.vega_time,
            status = EXCLUDED.status
            ;`,
		a.ID,
		a.Name,
		a.Symbol,
		a.Decimals,
		a.Quantum,
		a.Source,
		a.ERC20Contract,
		a.LifetimeLimit,
		a.WithdrawThreshold,
		a.TxHash,
		a.VegaTime,
		a.Status,
	)
	if err != nil {
		return err
	}

	as.AfterCommit(ctx, func() {
		// delete cache
		as.cacheLock.Lock()
		defer as.cacheLock.Unlock()
		delete(as.cache, a.ID.String())
	})
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
		getAssetQuery(ctx)+` WHERE id=$1`,
		entities.AssetID(id))

	if err == nil {
		as.cache[id] = a
	}
	return a, as.wrapE(err)
}

func (as *Assets) GetAll(ctx context.Context) ([]entities.Asset, error) {
	assets := []entities.Asset{}
	defer metrics.StartSQLQuery("Assets", "GetAll")()
	err := pgxscan.Select(ctx, as.Connection, &assets, getAssetQuery(ctx))
	return assets, err
}

func (as *Assets) GetAllWithCursorPagination(ctx context.Context, pagination entities.CursorPagination) (
	[]entities.Asset, entities.PageInfo, error,
) {
	var assets []entities.Asset
	var pageInfo entities.PageInfo
	var args []interface{}
	var err error

	query := getAssetQuery(ctx)
	query, args, err = PaginateQuery[entities.AssetCursor](query, args, assetOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}
	defer metrics.StartSQLQuery("Assets", "GetAllWithCursorPagination")()

	if err = pgxscan.Select(ctx, as.Connection, &assets, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get assets: %w", err)
	}

	assets, pageInfo = entities.PageEntities[*v2.AssetEdge](assets, pagination)

	return assets, pageInfo, nil
}

func getAssetQuery(ctx context.Context) string {
	return `SELECT * FROM assets_current`
}
