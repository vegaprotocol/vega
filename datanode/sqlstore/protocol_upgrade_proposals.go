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
