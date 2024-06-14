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
	_, err := as.Exec(ctx,
		`INSERT INTO assets(id, name, symbol, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, tx_hash, vega_time, status, chain_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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
            status = EXCLUDED.status,
            chain_id = EXCLUDED.chain_id
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
		a.ChainID,
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
	err := pgxscan.Get(ctx, as.ConnectionSource, &a,
		getAssetQuery(ctx)+` WHERE id=$1`,
		entities.AssetID(id))

	if err == nil {
		as.cache[id] = a
	}
	return a, as.wrapE(err)
}

func (as *Assets) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Asset, error) {
	defer metrics.StartSQLQuery("Assets", "GetByTxHash")()

	var assets []entities.Asset
	err := pgxscan.Select(ctx, as.ConnectionSource, &assets, `SELECT * FROM assets WHERE tx_hash=$1`, txHash)
	if err != nil {
		return nil, as.wrapE(err)
	}

	return assets, nil
}

func (as *Assets) GetAll(ctx context.Context) ([]entities.Asset, error) {
	assets := []entities.Asset{}
	defer metrics.StartSQLQuery("Assets", "GetAll")()
	err := pgxscan.Select(ctx, as.ConnectionSource, &assets, getAssetQuery(ctx))
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

	if err = pgxscan.Select(ctx, as.ConnectionSource, &assets, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get assets: %w", err)
	}

	assets, pageInfo = entities.PageEntities[*v2.AssetEdge](assets, pagination)

	return assets, pageInfo, nil
}

func getAssetQuery(_ context.Context) string {
	return `SELECT * FROM assets_current`
}
