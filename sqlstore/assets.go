package sqlstore

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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
	err := pgxscan.Select(ctx, as.Connection, &assets, `
		SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, lifetime_limit, withdraw_threshold, vega_time
		FROM assets`)
	return assets, err
}
