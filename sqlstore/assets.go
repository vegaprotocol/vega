package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Assets struct {
	*ConnectionSource
}

func NewAssets(connectionSource *ConnectionSource) *Assets {
	a := &Assets{
		ConnectionSource: connectionSource,
	}
	return a
}

func (as *Assets) Add(ctx context.Context, a entities.Asset) error {
	_, err := as.Connection.Exec(ctx,
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

	err := pgxscan.Get(ctx, as.Connection, &a,
		`SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, vega_time
		 FROM assets WHERE id=$1`,
		entities.NewAssetID(id))
	return a, err
}

func (as *Assets) GetAll(ctx context.Context) ([]entities.Asset, error) {

	assets := []entities.Asset{}
	err := pgxscan.Select(ctx, as.Connection, &assets, `
		SELECT id, name, symbol, total_supply, decimals, quantum, source, erc20_contract, vega_time
		FROM assets`)
	return assets, err
}
