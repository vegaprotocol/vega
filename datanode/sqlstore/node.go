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
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var ErrNodeNotFound = errors.New("node not found")

type Node struct {
	*ConnectionSource
}

func NewNode(connectionSource *ConnectionSource) *Node {
	return &Node{
		ConnectionSource: connectionSource,
	}
}

func (store *Node) UpsertNode(ctx context.Context, node *entities.Node) error {
	defer metrics.StartSQLQuery("Node", "UpsertNode")()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO nodes (
			id,
			vega_pub_key,
			tendermint_pub_key,
			ethereum_address,
			info_url,
			location,
			status,
			name,
			avatar_url,
			vega_time)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE
		SET
			vega_pub_key = EXCLUDED.vega_pub_key,
			tendermint_pub_key = EXCLUDED.tendermint_pub_key,
			ethereum_address = EXCLUDED.ethereum_address,
			info_url = EXCLUDED.info_url,
			location = EXCLUDED.location,
			status = EXCLUDED.status,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url`,
		node.ID,
		node.PubKey,
		node.TmPubKey,
		node.EthereumAddress,
		node.InfoUrl,
		node.Location,
		node.Status,
		node.Name,
		node.AvatarUrl,
		node.VegaTime,
	)

	return err
}

// AddNodeAnnoucedEvent store data about which epoch a particular node was added or removed from the roster of alidators
func (store *Node) AddNodeAnnoucedEvent(ctx context.Context, nodeID entities.NodeID, vegatime time.Time, aux *entities.ValidatorUpdateAux) error {
	defer metrics.StartSQLQuery("Node", "AddNodeAnnoucedEvent")()
	_, err := store.pool.Exec(ctx, `
		INSERT INTO nodes_announced (
			node_id,
			epoch_seq,
			added,
		    vega_time)
		VALUES
			($1, $2, $3, $4)
		ON CONFLICT (node_id, epoch_seq, vega_time) DO UPDATE SET
			added=EXCLUDED.added`,
		nodeID,
		aux.FromEpoch,
		aux.Added,
		vegatime,
	)

	return err
}

func (store *Node) UpsertRanking(ctx context.Context, rs *entities.RankingScore, aux *entities.RankingScoreAux) error {
	defer metrics.StartSQLQuery("Node", "UpsertRanking")()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO ranking_scores (
			node_id,
			epoch_seq,
			stake_score,
			performance_score,
			ranking_score,
			voting_power,
			previous_status,
			status,
			vega_time)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		aux.NodeId,
		rs.EpochSeq,
		rs.StakeScore,
		rs.PerformanceScore,
		rs.RankingScore,
		rs.VotingPower,
		rs.PreviousStatus,
		rs.Status,
		rs.VegaTime,
	)

	return err
}

func (store *Node) UpsertScore(ctx context.Context, rs *entities.RewardScore, aux *entities.RewardScoreAux) error {
	defer metrics.StartSQLQuery("Node", "UpsertScore")()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO reward_scores (
			node_id,
			epoch_seq,
			validator_node_status,
			raw_validator_score,
			performance_score,
			multisig_score,
			validator_score,
			normalised_score,
			vega_time) 
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		aux.NodeId,
		rs.EpochSeq,
		rs.ValidatorNodeStatus,
		rs.RawValidatorScore,
		rs.PerformanceScore,
		rs.MultisigScore,
		rs.ValidatorScore,
		rs.NormalisedScore,
		rs.VegaTime,
	)

	return err
}

func (store *Node) UpdatePublicKey(ctx context.Context, kr *entities.KeyRotation) error {
	defer metrics.StartSQLQuery("Node", "UpdatePublicKey")()

	_, err := store.pool.Exec(ctx, `UPDATE nodes SET pub_key = $1 WHERE id = $2`, kr.NewPubKey, kr.NodeID)

	return err
}

func (store *Node) GetNodeData(ctx context.Context) (entities.NodeData, error) {
	defer metrics.StartSQLQuery("Node", "GetNodeData")()
	query := `
		WITH
			uptime AS (
				SELECT ` +
		// 			Uptime denominated in minutes, hence division by 60 seconds
		`			EXTRACT(EPOCH FROM SUM(end_time - start_time)) / 60.0 AS total,
					ROW_NUMBER() OVER () AS id
				FROM
					epochs
				WHERE` +
		// 			Only include epochs that have elapsed
		`			end_time IS NOT NULL
			),

			staked AS (
				SELECT
					COALESCE(SUM(amount),0) AS total,
					ROW_NUMBER() OVER () AS id
				FROM
					delegations
				WHERE
					-- Select the current epoch
					epoch_id = (SELECT MAX(id) FROM epochs)
			),
			/* partitioned by node_id find the join/leave annoucement with the biggest epoch that is also less or equal to the target epoch */
			join_event AS (
				SELECT 
					node_id, added 
				FROM 
					( 
						SELECT 
							node_id, added, epoch_seq, Row_Number() 
						OVER(PARTITION BY node_id order BY epoch_seq desc) 
						AS 
							row_number 
						FROM 
							nodes_announced 
						WHERE 
							epoch_seq <= (SELECT MAX(id) FROM epochs)
					) AS a
				WHERE 
					row_number = 1 AND added = true
			),

			node_totals AS ( ` +
		// 		Currently there's no mechanism for changing the node status
		// 		and it's unclear what exactly an inactive node is
		`		SELECT
					COUNT(1) filter (where nodes.status = 'NODE_STATUS_VALIDATOR') 		AS validating_nodes,
					0 																	AS inactive_nodes,
					COUNT(1) filter (where nodes.status <> 'NODE_STATUS_UNSPECIFIED') 	AS total_nodes,
					ROW_NUMBER() OVER () AS id
				FROM
					nodes
					WHERE EXISTS (
						SELECT *
						FROM join_event WHERE node_id = nodes.id
					)

			)
		SELECT
			staked.total AS staked_total,
			uptime.total AS uptime,
			node_totals.validating_nodes,
			node_totals.inactive_nodes,
			node_totals.total_nodes

		FROM node_totals ` +
		// These joins are "fake" as to extract all the individual values as one row
		`JOIN staked ON node_totals.id = staked.id
		JOIN uptime ON uptime.id = staked.id;
	`

	nodeData := entities.NodeData{}

	err := pgxscan.Get(ctx, store.pool, &nodeData, query)
	return nodeData, err
}

func (store *Node) GetNodes(ctx context.Context, epochSeq uint64, pagination entities.CursorPagination) ([]entities.Node, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Node", "GetNodes")()
	var nodes []entities.Node
	var pageInfo entities.PageInfo

	query := `WITH
	current_delegations AS (
		SELECT * FROM delegations_current
		WHERE epoch_id = $1
	),
	pending_delegations AS (
		SELECT * FROM delegations_current
		WHERE epoch_id = $1 + 1
	),

	/* partitioned by node_id find the join/leave annoucement with the biggest epoch that is also less or equal to the target epoch */
	join_event AS (
		SELECT 
			node_id, added 
		FROM ( 
			SELECT 
				node_id, added, epoch_seq, Row_Number() 
			OVER(PARTITION BY node_id order BY epoch_seq desc) 
			AS 
				row_number 
			FROM 
				nodes_announced 
			WHERE epoch_seq <= $1) AS a
		WHERE row_number = 1 AND added = true
	),
	this_epoch AS (
		SELECT nodes.id AS node_id,
			COALESCE(SUM(current_delegations.amount) FILTER (WHERE current_delegations.party_id = nodes.vega_pub_key), 0) AS staked_by_operator,
			COALESCE(SUM(current_delegations.amount) FILTER (WHERE current_delegations.party_id != nodes.vega_pub_key), 0) AS staked_by_delegates,
			COALESCE(SUM(current_delegations.amount), 0) AS staked_total,
			COALESCE(JSON_AGG(JSON_BUILD_OBJECT(
				'party_id', ENCODE(current_delegations.party_id, 'hex'),
				'node_id', ENCODE(current_delegations.node_id, 'hex'),
				'epoch_id', current_delegations.epoch_id,
				'amount', current_delegations.amount)
			) FILTER (WHERE current_delegations.party_id IS NOT NULL), json_build_array()) AS "delegations"
		FROM nodes LEFT JOIN current_delegations  on current_delegations.node_id = nodes.id
		GROUP BY nodes.id),
	next_epoch AS (
		SELECT nodes.id as node_id,
		       COALESCE(SUM(pending_delegations.amount), 0) AS staked_total
		FROM nodes LEFT JOIN pending_delegations on pending_delegations.node_id = nodes.id
		GROUP BY nodes.id
	)

	SELECT
		nodes.id,
		nodes.vega_pub_key,
		nodes.tendermint_pub_key,
		nodes.ethereum_address,
		nodes.name,
		nodes.location,
		nodes.info_url,
		nodes.avatar_url,
		nodes.status,
		ROW_TO_JSON(reward_scores.*)::JSONB AS "reward_score",
		ROW_TO_JSON(ranking_scores.*)::JSONB AS "ranking_score",
		this_epoch.delegations,
		this_epoch.staked_by_operator,
		this_epoch.staked_by_delegates,
		this_epoch.staked_total,
		next_epoch.staked_total - this_epoch.staked_total as pending_stake
	FROM nodes
	JOIN this_epoch on nodes.id = this_epoch.node_id
	JOIN next_epoch on nodes.id = next_epoch.node_id
	LEFT JOIN ranking_scores ON ranking_scores.node_id = nodes.id AND ranking_scores.epoch_seq = $1
	LEFT JOIN reward_scores ON reward_scores.node_id = nodes.id AND reward_scores.epoch_seq = $1
	WHERE EXISTS (
		SELECT *
		FROM join_event WHERE node_id = nodes.id
		)
	`
	args := []interface{}{
		epochSeq,
	}

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("id", sorting, cmp, entities.NewNodeID(cursor)),
	}

	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	if err := pgxscan.Select(ctx, store.pool, &nodes, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get nodes: %w", err)
	}

	nodes, pageInfo = entities.PageEntities[*v2.NodeEdge](nodes, pagination)

	return nodes, pageInfo, nil
}

func (store *Node) GetNodeByID(ctx context.Context, nodeId string, epochSeq uint64) (entities.Node, error) {
	defer metrics.StartSQLQuery("Node", "GetNodeById")()

	var node entities.Node
	id := entities.NewNodeID(nodeId)

	query := `WITH
	current_delegations AS (
		SELECT * FROM delegations_current
		WHERE epoch_id = $1
	),
	pending_delegations AS (
		SELECT * FROM delegations_current
		WHERE epoch_id = $1 + 1
	),

	/* partitioned by node_id find the join/leave annoucement with the biggest epoch that is also less or equal to the target epoch */
	join_event AS (
		SELECT 
			node_id, added 
		FROM ( 
			SELECT 
				node_id, added, epoch_seq, Row_Number() 
			OVER(PARTITION BY node_id order BY epoch_seq desc) 
			AS 
				row_number 
			FROM 
				nodes_announced 
			WHERE epoch_seq <= $1) AS a
		WHERE row_number = 1 AND added = true
	),
	this_epoch AS (
		SELECT nodes.id AS node_id,
			COALESCE(SUM(current_delegations.amount) FILTER (WHERE current_delegations.party_id = nodes.vega_pub_key), 0) AS staked_by_operator,
			COALESCE(SUM(current_delegations.amount) FILTER (WHERE current_delegations.party_id != nodes.vega_pub_key), 0) AS staked_by_delegates,
			COALESCE(SUM(current_delegations.amount), 0) AS staked_total,
			COALESCE(JSON_AGG(JSON_BUILD_OBJECT(
				'party_id', ENCODE(current_delegations.party_id, 'hex'),
				'node_id', ENCODE(current_delegations.node_id, 'hex'),
				'epoch_id', current_delegations.epoch_id,
				'amount', current_delegations.amount)
			) FILTER (WHERE current_delegations.party_id IS NOT NULL), json_build_array()) AS "delegations"
		FROM nodes LEFT JOIN current_delegations  on current_delegations.node_id = nodes.id
		GROUP BY nodes.id),
	next_epoch AS (
		SELECT nodes.id as node_id,
		       COALESCE(SUM(pending_delegations.amount), 0) AS staked_total
		FROM nodes LEFT JOIN pending_delegations on pending_delegations.node_id = nodes.id
		GROUP BY nodes.id
	)

	SELECT
		nodes.id,
		nodes.vega_pub_key,
		nodes.tendermint_pub_key,
		nodes.ethereum_address,
		nodes.name,
		nodes.location,
		nodes.info_url,
		nodes.avatar_url,
		nodes.status,
		ROW_TO_JSON(reward_scores.*)::JSONB AS "reward_score",
		ROW_TO_JSON(ranking_scores.*)::JSONB AS "ranking_score",
		this_epoch.delegations,
		this_epoch.staked_by_operator,
		this_epoch.staked_by_delegates,
		this_epoch.staked_total,
		next_epoch.staked_total - this_epoch.staked_total as pending_stake
	FROM nodes
	JOIN this_epoch on nodes.id = this_epoch.node_id
	JOIN next_epoch on nodes.id = next_epoch.node_id
	LEFT JOIN ranking_scores ON ranking_scores.node_id = nodes.id AND ranking_scores.epoch_seq = $1
	LEFT JOIN reward_scores ON reward_scores.node_id = nodes.id AND reward_scores.epoch_seq = $1
	WHERE EXISTS (
		SELECT *
		FROM join_event WHERE node_id = nodes.id
		)
		AND nodes.id=$2
	`

	err := pgxscan.Get(ctx, store.pool, &node, query, epochSeq, id)
	if pgxscan.NotFound(err) {
		return node, fmt.Errorf("'%v': %w", nodeId, ErrNodeNotFound)
	}
	return node, err
}
