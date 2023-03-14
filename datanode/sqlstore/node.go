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
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var nodeOrdering = TableOrdering{
	ColumnOrdering{Name: "id", Sorting: ASC},
}

type Node struct {
	*ConnectionSource
}

func NewNode(connectionSource *ConnectionSource) *Node {
	return &Node{
		ConnectionSource: connectionSource,
	}
}

// this query requires a epoch_id as a first argument: WHERE epoch_id = $1
func selectNodeQuery() string {
	return `WITH
	current_delegations AS (
		SELECT * FROM delegations_current
		WHERE epoch_id = $1
	),
	pending_delegations AS (
		SELECT * FROM delegations_current
		WHERE epoch_id = $1 + 1
	),

	/* partitioned by node_id find the join/leave announcement with the biggest epoch that is also less or equal to the target epoch */
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
		)`
}

func (store *Node) UpsertNode(ctx context.Context, node *entities.Node) error {
	defer metrics.StartSQLQuery("Node", "UpsertNode")()

	_, err := store.Connection.Exec(ctx, `
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
			tx_hash,
			vega_time)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE
		SET
			vega_pub_key = EXCLUDED.vega_pub_key,
			tendermint_pub_key = EXCLUDED.tendermint_pub_key,
			ethereum_address = EXCLUDED.ethereum_address,
			info_url = EXCLUDED.info_url,
			location = EXCLUDED.location,
			status = EXCLUDED.status,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			tx_hash = EXCLUDED.tx_hash,
			vega_time = EXCLUDED.vega_time`,
		node.ID,
		node.PubKey,
		node.TmPubKey,
		node.EthereumAddress,
		node.InfoURL,
		node.Location,
		node.Status,
		node.Name,
		node.AvatarURL,
		node.TxHash,
		node.VegaTime,
	)

	return err
}

// AddNodeAnnouncedEvent store data about which epoch a particular node was added or removed from the roster of validators.
func (store *Node) AddNodeAnnouncedEvent(ctx context.Context, nodeID string, vegatime time.Time, aux *entities.ValidatorUpdateAux) error {
	defer metrics.StartSQLQuery("Node", "AddNodeAnnouncedEvent")()
	_, err := store.Connection.Exec(ctx, `
		INSERT INTO nodes_announced (
			node_id,
			epoch_seq,
			added,
			tx_hash,
		    vega_time)
		VALUES
			($1, $2, $3, $4, $5)
		ON CONFLICT (node_id, epoch_seq, vega_time) DO UPDATE SET
			added=EXCLUDED.added`,
		entities.NodeID(nodeID),
		aux.EpochSeq,
		aux.Added,
		aux.TxHash,
		vegatime,
	)

	return err
}

func (store *Node) UpsertRanking(ctx context.Context, rs *entities.RankingScore, aux *entities.RankingScoreAux) error {
	defer metrics.StartSQLQuery("Node", "UpsertRanking")()

	_, err := store.Connection.Exec(ctx, `
		INSERT INTO ranking_scores (
			node_id,
			epoch_seq,
			stake_score,
			performance_score,
			ranking_score,
			voting_power,
			previous_status,
			status,
			tx_hash,
			vega_time)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (node_id, epoch_seq) DO UPDATE
		SET
			stake_score = EXCLUDED.stake_score,
		    performance_score = EXCLUDED.performance_score,
		    ranking_score = EXCLUDED.ranking_score,
		    voting_power = EXCLUDED.voting_power,
		    previous_status = EXCLUDED.previous_status,
		    status = EXCLUDED.status,
		    tx_hash = EXCLUDED.tx_hash,
		    vega_time = EXCLUDED.vega_time`,
		aux.NodeID,
		rs.EpochSeq,
		rs.StakeScore,
		rs.PerformanceScore,
		rs.RankingScore,
		rs.VotingPower,
		rs.PreviousStatus,
		rs.Status,
		rs.TxHash,
		rs.VegaTime,
	)

	return err
}

func (store *Node) UpsertScore(ctx context.Context, rs *entities.RewardScore, aux *entities.RewardScoreAux) error {
	defer metrics.StartSQLQuery("Node", "UpsertScore")()

	_, err := store.Connection.Exec(ctx, `
		INSERT INTO reward_scores (
			node_id,
			epoch_seq,
			validator_node_status,
			raw_validator_score,
			performance_score,
			multisig_score,
			validator_score,
			normalised_score,
			tx_hash,
			vega_time)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		aux.NodeID,
		rs.EpochSeq,
		rs.ValidatorNodeStatus,
		rs.RawValidatorScore,
		rs.PerformanceScore,
		rs.MultisigScore,
		rs.ValidatorScore,
		rs.NormalisedScore,
		rs.TxHash,
		rs.VegaTime,
	)

	return err
}

func (store *Node) UpdatePublicKey(ctx context.Context, kr *entities.KeyRotation) error {
	defer metrics.StartSQLQuery("Node", "UpdatePublicKey")()

	_, err := store.Connection.Exec(ctx, `UPDATE nodes SET vega_pub_key = $1 WHERE id = $2`, kr.NewPubKey, kr.NodeID)

	return err
}

func (store *Node) UpdateEthereumAddress(ctx context.Context, kr entities.EthereumKeyRotation) error {
	defer metrics.StartSQLQuery("Node", "UpdateEthereumPublicKey")()

	_, err := store.Connection.Exec(ctx, `UPDATE nodes SET ethereum_address = $1 WHERE id = $2`, kr.NewAddress, kr.NodeID)

	return err
}

func (store *Node) GetNodeData(ctx context.Context, epochSeq uint64) (entities.NodeData, error) {
	defer metrics.StartSQLQuery("Node", "GetNetworkData")()
	query := `
		WITH
			uptime AS (
				SELECT
		-- 			Uptime denominated in minutes, hence division by 60 seconds
					EXTRACT(EPOCH FROM SUM(end_time - start_time)) / 60.0 AS total,
					ROW_NUMBER() OVER () AS id
				FROM
					epochs
				WHERE
					id <= $1
			),

			staked AS (
				SELECT
					COALESCE(SUM(amount),0) AS total,
					ROW_NUMBER() OVER () AS id
				FROM
					delegations
				WHERE
					-- Select the current epoch
					epoch_id = $1
			)
		SELECT
			staked.total AS staked_total,
			coalesce(uptime.total, 0) AS uptime

		FROM staked
		--  This join is "fake" as to extract all the individual values as one row
		JOIN uptime ON uptime.id = staked.id;
	`

	nodeData := entities.NodeData{}
	err := pgxscan.Get(ctx, store.Connection, &nodeData, query, epochSeq)
	if err != nil {
		return nodeData, store.wrapE(err)
	}

	// now we fill in the more complicated bits about node sets
	nodes, _, err := store.GetNodes(ctx, epochSeq, entities.DefaultCursorPagination(true))
	if err != nil {
		return nodeData, err
	}

	nodeSets := map[entities.ValidatorNodeStatus]*entities.NodeSet{
		entities.ValidatorNodeStatusTendermint: &nodeData.TendermintNodes,
		entities.ValidatorNodeStatusErsatz:     &nodeData.ErsatzNodes,
		entities.ValidatorNodeStatusPending:    &nodeData.PendingNodes,
	}
	for _, n := range nodes {
		if n.RankingScore == nil {
			store.log.Warn("ignoring node with empty ranking score", logging.String("id", n.ID.String()))
			continue
		}
		status := n.RankingScore.Status
		previousStatus := n.RankingScore.PreviousStatus
		if status == entities.ValidatorNodeStatusUnspecified {
			continue
		}
		ns := nodeSets[status]
		nodeData.TotalNodes++
		ns.Total++

		// but was it active
		if n.RankingScore.PerformanceScore.IsZero() {
			ns.Inactive++
			nodeData.InactiveNodes++
		}

		// check if the node was promoted or demoted into its set this epoch
		switch {
		case uint32(status) < uint32(previousStatus):
			ns.Promoted = append(ns.Promoted, n.ID.String())
		case uint32(status) > uint32(previousStatus):
			ns.Demoted = append(ns.Promoted, n.ID.String())
		default:
			// node stayed in the same set, thats cool
		}
	}
	return nodeData, err
}

func (store *Node) GetNodes(ctx context.Context, epochSeq uint64, pagination entities.CursorPagination) ([]entities.Node, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Node", "GetNodes")()
	var (
		nodes    []entities.Node
		pageInfo entities.PageInfo
		err      error
	)

	args := []interface{}{
		epochSeq,
	}

	query, args, err := PaginateQuery[entities.NodeCursor](selectNodeQuery(), args, nodeOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err = pgxscan.Select(ctx, store.Connection, &nodes, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get nodes: %w", err)
	}

	nodes, pageInfo = entities.PageEntities[*v2.NodeEdge](nodes, pagination)

	return nodes, pageInfo, nil
}

func (store *Node) GetNodeByID(ctx context.Context, nodeID string, epochSeq uint64) (entities.Node, error) {
	defer metrics.StartSQLQuery("Node", "GetNodeById")()

	var node entities.Node
	id := entities.NodeID(nodeID)

	query := fmt.Sprintf("%s AND nodes.id=$2", selectNodeQuery())
	return node, store.wrapE(pgxscan.Get(ctx, store.Connection, &node, query, epochSeq, id))
}

func (store *Node) GetNodeTxHash(ctx context.Context, nodeID string, epochSeq uint64) (entities.Node, error) {
	defer metrics.StartSQLQuery("Node", "GetNodeById")()

	var node entities.Node
	id := entities.NodeID(nodeID)

	query := fmt.Sprintf("%s AND nodes.id=$2", selectNodeQuery())
	return node, store.wrapE(pgxscan.Get(ctx, store.Connection, &node, query, epochSeq, id))
}
