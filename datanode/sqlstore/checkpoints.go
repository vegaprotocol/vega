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

func (c *Checkpoints) Add(ctx context.Context, r entities.Checkpoint) error {
	defer metrics.StartSQLQuery("Checkpoints", "Add")()
	_, err := c.Connection.Exec(ctx,
		`INSERT INTO checkpoints(
			hash,
			block_hash,
			block_height,
			tx_hash,
			vega_time,
			seq_num)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 `,
		r.Hash, r.BlockHash, r.BlockHeight, r.TxHash, r.VegaTime, r.SeqNum)
	return err
}

func (c *Checkpoints) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.Checkpoint, entities.PageInfo, error) {
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

	if err = pgxscan.Select(ctx, c.Connection, &nps, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get checkpoint data: %w", err)
	}

	nps, pageInfo = entities.PageEntities[*v2.CheckpointEdge](nps, pagination)

	return nps, pageInfo, nil
}
