package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Assets struct {
	*SQLStore
}

func NewAssets(sqlStore *SQLStore) *Assets {
	a := &Assets{
		SQLStore: sqlStore,
	}
	return a
}

func (as *Assets) Add(a entities.Asset) error {
	ctx := context.Background()
	_, err := as.pool.Exec(ctx,
		`INSERT INTO assets(id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, vega_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID,
		a.Name,
		a.Symbol,
		a.TotalSupply,
		a.Decimals,
		a.Quantum,
		a.Source,
		a.ERC20Contract,
		a.VegaTime)
	return err
}

func (as *Assets) GetByID(ctx context.Context, id string) (entities.Asset, error) {
	a := entities.Asset{}

	idBytes := entities.MakeAssetID(id)

	ctxTimeout, cancel := context.WithTimeout(ctx, as.conf.Timeout.Get())
	defer cancel()

	err := pgxscan.Get(ctxTimeout, as.pool, &a,
		`SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, vega_time
		 FROM assets WHERE id=$1`,
		idBytes)
	return a, err
}

func (as *Assets) GetAll(ctx context.Context) ([]entities.Asset, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, as.conf.Timeout.Get())
	defer cancel()

	assets := []entities.Asset{}
	err := pgxscan.Select(ctxTimeout, as.pool, &assets, `
		SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, vega_time
		FROM assets`)
	return assets, err
}
