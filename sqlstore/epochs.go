package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Epochs struct {
	*SQLStore
}

func NewEpochs(sqlStore *SQLStore) *Epochs {
	e := &Epochs{
		SQLStore: sqlStore,
	}
	return e
}

func (es *Epochs) Add(ctx context.Context, r entities.Epoch) error {
	_, err := es.pool.Exec(ctx,
		`INSERT INTO epochs(
			id,
			start_time,
			expire_time,
			end_time,
			vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5)
		 ON CONFLICT (id, vega_time)
		 DO UPDATE SET start_time=EXCLUDED.start_time,
		 	           expire_time=EXCLUDED.expire_time,
		               end_time=EXCLUDED.end_time
		 ;`,
		r.ID, r.StartTime, r.ExpireTime, r.EndTime, r.VegaTime)
	return err
}

func (rs *Epochs) GetAll(ctx context.Context) ([]entities.Epoch, error) {
	epochs := []entities.Epoch{}
	err := pgxscan.Select(ctx, rs.pool, &epochs, `
		SELECT DISTINCT ON (id) * from epochs ORDER BY id, vega_time desc;`)
	return epochs, err
}

func (rs *Epochs) Get(ctx context.Context, ID int64) (entities.Epoch, error) {
	query := `SELECT DISTINCT ON (id) * FROM epochs WHERE id=$1 ORDER BY id, vega_time desc;`

	epoch := entities.Epoch{}
	err := pgxscan.Get(ctx, rs.pool, &epoch, query, ID)
	if err != nil {
		return entities.Epoch{}, fmt.Errorf("querying epochs: %w", err)
	}
	return epoch, nil
}

func (rs *Epochs) GetCurrent(ctx context.Context) (entities.Epoch, error) {
	query := `SELECT * FROM epochs ORDER BY id desc, vega_time desc FETCH FIRST ROW ONLY;`

	epoch := entities.Epoch{}
	err := pgxscan.Get(ctx, rs.pool, &epoch, query)
	if err != nil {
		return entities.Epoch{}, fmt.Errorf("querying epocs: %w", err)
	}
	return epoch, nil
}
