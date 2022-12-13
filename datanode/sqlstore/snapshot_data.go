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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type CoreSnapshotData struct {
	*ConnectionSource
}

var snapshotOrdering = TableOrdering{
	ColumnOrdering{Name: "block_height", Sorting: DESC},
}

func NewCoreSnapshotData(connectionSource *ConnectionSource) *CoreSnapshotData {
	return &CoreSnapshotData{ConnectionSource: connectionSource}
}

func (s *CoreSnapshotData) Add(ctx context.Context, csd entities.CoreSnapshotData) error {
	defer metrics.StartSQLQuery("CoreSnapshotData", "Add")()

	_, err := s.Connection.Exec(ctx,
		`INSERT INTO core_snapshots(
			block_height,
			block_hash,
			vega_core_version,
			vega_time,
			tx_hash)
		 VALUES ($1,  $2,  $3,  $4, $5);`,
		csd.BlockHeight, csd.BlockHash, csd.VegaCoreVersion, csd.VegaTime, csd.TxHash)
	return err
}

func (s *CoreSnapshotData) List(ctx context.Context, pagination entities.CursorPagination) ([]entities.CoreSnapshotData, entities.PageInfo, error) {
	args := []interface{}{}
	query := `
        SELECT block_height,
               block_hash,
			   vega_core_version,
               vega_time,
               tx_hash
        FROM core_snapshots
	`
	var err error

	pageInfo := entities.PageInfo{}
	query, args, err = PaginateQuery[entities.CoreSnapshotDataCursor](query, args, snapshotOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	defer metrics.StartSQLQuery("CoreSnapshotData", "List")()
	snaps := make([]entities.CoreSnapshotData, 0)
	if err := pgxscan.Select(ctx, s.Connection, &snaps, query, args...); err != nil {
		return snaps, pageInfo, err
	}

	snaps, pageInfo = entities.PageEntities[*v2.CoreSnapshotEdge](snaps, pagination)
	return snaps, pageInfo, nil
}
