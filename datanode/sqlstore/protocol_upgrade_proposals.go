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
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type ProtocolUpgradeProposals struct {
	*ConnectionSource
}

var pupOrdering = TableOrdering{
	ColumnOrdering{Name: "upgrade_block_height", Sorting: ASC},
	ColumnOrdering{Name: "vega_release_tag", Sorting: ASC},
}

func NewProtocolUpgradeProposals(connectionSource *ConnectionSource) *ProtocolUpgradeProposals {
	p := &ProtocolUpgradeProposals{
		ConnectionSource: connectionSource,
	}
	return p
}

func (ps *ProtocolUpgradeProposals) Add(ctx context.Context, p entities.ProtocolUpgradeProposal) error {
	defer metrics.StartSQLQuery("ProtocolUpgradeProposals", "Add")()
	if p.Approvers == nil {
		p.Approvers = []string{}
	}

	_, err := ps.Connection.Exec(ctx,
		`INSERT INTO protocol_upgrade_proposals(
			upgrade_block_height,
			vega_release_tag,
			approvers,
			status,
			vega_time,
			tx_hash)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6)
		 ON CONFLICT (vega_time, upgrade_block_height, vega_release_tag) DO UPDATE SET
			approvers = EXCLUDED.approvers,
			status = EXCLUDED.status,
			tx_hash = EXCLUDED.tx_hash;
		`,
		p.UpgradeBlockHeight, p.VegaReleaseTag, p.Approvers, p.Status, p.VegaTime, p.TxHash)
	return err
}

func (ps *ProtocolUpgradeProposals) List(ctx context.Context,
	status *entities.ProtocolUpgradeProposalStatus,
	approvedBy *string,
	pagination entities.CursorPagination,
) ([]entities.ProtocolUpgradeProposal, entities.PageInfo, error) {
	args := []interface{}{}
	query := `
        SELECT upgrade_block_height,
               vega_release_tag,
               approvers,
               status,
               vega_time,
               tx_hash
        FROM protocol_upgrade_proposals_current
	`
	var predicates []string
	var err error

	if status != nil {
		predicates = append(predicates, fmt.Sprintf("status=%s", nextBindVar(&args, *status)))
	}

	if approvedBy != nil {
		predicates = append(predicates, fmt.Sprintf("%s=ANY(approvers)", nextBindVar(&args, *approvedBy)))
	}

	if len(predicates) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(predicates, " AND "))
	}

	pageInfo := entities.PageInfo{}
	query, args, err = PaginateQuery[entities.ProtocolUpgradeProposalCursor](query, args, pupOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	defer metrics.StartSQLQuery("ProtocolUpgradeProposals", "List")()
	pups := make([]entities.ProtocolUpgradeProposal, 0)
	if err := pgxscan.Select(ctx, ps.Connection, &pups, query, args...); err != nil {
		return pups, pageInfo, err
	}

	pups, pageInfo = entities.PageEntities[*v2.ProtocolUpgradeProposalEdge](pups, pagination)
	return pups, pageInfo, nil
}

func (ps *ProtocolUpgradeProposals) GetByTxHash(
	ctx context.Context,
	txHash entities.TxHash,
) ([]entities.ProtocolUpgradeProposal, error) {
	defer metrics.StartSQLQuery("ProtocolUpgradeProposals", "GetByTxHash")()

	var pups []entities.ProtocolUpgradeProposal
	query := `SELECT upgrade_block_height, vega_release_tag, approvers, status, vega_time, tx_hash
		FROM protocol_upgrade_proposals WHERE tx_hash = $1`

	if err := pgxscan.Select(ctx, ps.Connection, &pups, query, txHash); err != nil {
		return nil, err
	}

	return pups, nil
}
