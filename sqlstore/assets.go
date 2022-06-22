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
		`INSERT INTO assets(id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, vega_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
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
		a.VegaTime)
	return err
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
		`SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, vega_time
		 FROM assets WHERE id=$1`,
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
	return `SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, vega_time
		FROM assets`
}
