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
	"fmt"
	"strconv"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
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

func (np *Checkpoints) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.Checkpoint, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Checkpoints", "GetAll")()
	var nps []entities.Checkpoint
	var pageInfo entities.PageInfo
	query := `SELECT * FROM checkpoints`

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	var blockHeight int64

	if cursor != "" {
		var err error
		if blockHeight, err = strconv.ParseInt(cursor, 10, 64); err != nil {
			return nil, pageInfo, fmt.Errorf("invalid cursor value: %w", err)
		}
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("block_height", sorting, cmp, blockHeight),
	}

	var args []interface{}
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	if err := pgxscan.Select(ctx, np.Connection, &nps, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get checkpoint data: %w", err)
	}

	nps, pageInfo = entities.PageEntities[*v2.CheckpointEdge](nps, pagination)

	return nps, pageInfo, nil
}
