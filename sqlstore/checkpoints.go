package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Checkpoints struct {
	*ConnectionSource
}

func NewCheckpoints(connectionSource *ConnectionSource) *Checkpoints {
	p := &Checkpoints{
		ConnectionSource: connectionSource,
	}
	return p
}

func (ps *Checkpoints) Add(ctx context.Context, r entities.Checkpoint) error {
	_, err := ps.Connection.Exec(ctx,
		`INSERT INTO checkpoints(
			hash,
			block_hash,
			block_height,
			vega_time)
		 VALUES ($1, $2, $3, $4)
		 `,
		r.Hash, r.BlockHash, r.BlockHeight, r.VegaTime)
	return err
}

func (np *Checkpoints) GetAll(ctx context.Context) ([]entities.Checkpoint, error) {
	var nps []entities.Checkpoint
	query := `SELECT * FROM checkpoints ORDER BY block_height DESC`
	err := pgxscan.Select(ctx, np.Connection, &nps, query)
	return nps, err
}
