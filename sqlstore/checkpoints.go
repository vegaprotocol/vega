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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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
	defer metrics.StartSQLQuery("Checkpoints", "Add")()
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
	defer metrics.StartSQLQuery("Checkpoints", "GetAll")()
	var nps []entities.Checkpoint
	query := `SELECT * FROM checkpoints ORDER BY block_height DESC`
	err := pgxscan.Select(ctx, np.Connection, &nps, query)
	return nps, err
}
