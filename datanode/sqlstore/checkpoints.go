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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type Checkpoints struct {
	*ConnectionSource
}

var checkpointOrdering = TableOrdering{
	ColumnOrdering{Name: "block_height", Sorting: ASC},
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
			tx_hash,
			vega_time)
		 VALUES ($1, $2, $3, $4, $5)
		 `,
		r.Hash, r.BlockHash, r.BlockHeight, r.TxHash, r.VegaTime)
	return err
}

func (np *Checkpoints) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.Checkpoint, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Checkpoints", "GetAll")()
	var nps []entities.Checkpoint
	var pageInfo entities.PageInfo
	var err error

	query := `SELECT * FROM checkpoints`
	var args []interface{}
	query, args, err = PaginateQuery[entities.CheckpointCursor](query, args, checkpointOrdering, pagination)
	if err != nil {
		return nps, pageInfo, err
	}

	if err = pgxscan.Select(ctx, np.Connection, &nps, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get checkpoint data: %w", err)
	}

	nps, pageInfo = entities.PageEntities[*v2.CheckpointEdge](nps, pagination)

	return nps, pageInfo, nil
}
