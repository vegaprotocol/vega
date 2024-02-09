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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type CoreSnapshotData struct {
	*ConnectionSource
}

var snapshotOrdering = TableOrdering{
	// ASC here actually means DESC for some reason
	ColumnOrdering{Name: "block_height", Sorting: ASC},
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
		 VALUES ($1,  $2,  $3,  $4, $5)`,
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
