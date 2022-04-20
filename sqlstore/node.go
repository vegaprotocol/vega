package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

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

func (store *Node) UpsertRanking(ctx context.Context, rs *entities.RankingScore, aux *entities.RankingScoreAux) error {
	defer metrics.StartSQLQuery("Node", "UpsertRanking")()

	_, err := store.pool.Exec(ctx, `UPDATE nodes SET ranking_score = $1 WHERE id = $2`, rs, aux.NodeId)

	return err
}

func (store *Node) UpsertScore(ctx context.Context, rs *entities.RewardScore, aux *entities.RewardScoreAux) error {
	defer metrics.StartSQLQuery("Node", "UpsertScore")()

	_, err := store.pool.Exec(ctx, `UPDATE nodes SET reward_score = $1 WHERE id = $2`, rs, aux.NodeId)

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
					SUM(amount) AS total,
					ROW_NUMBER() OVER () AS id
				FROM
					delegations
				WHERE
					-- Select the current epoch
					epoch_id = (SELECT MAX(id) FROM epochs)
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

func (store *Node) GetNodes(ctx context.Context) ([]entities.Node, error) {
	defer metrics.StartSQLQuery("Node", "GetNodes")()
	var nodes []entities.Node

	query := `WITH 
	current_epoch AS (
		SELECT MAX(id) AS id FROM epochs
	),
	delegations AS (
		SELECT * FROM delegations
		WHERE epoch_id = (SELECT id FROM current_epoch)
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
	
		(CASE WHEN (nodes.reward_score->>'EpochSeq')::BIGINT = (SELECT id FROM current_epoch) THEN nodes.reward_score END) AS reward_score,
		(CASE WHEN (nodes.ranking_score->>'EpochSeq')::BIGINT = (SELECT id FROM current_epoch) THEN nodes.ranking_score END) AS ranking_score,
		
		COALESCE(SUM(delegations.amount) FILTER (WHERE delegations.party_id = nodes.vega_pub_key), 0) AS staked_by_operator,
		COALESCE(SUM(delegations.amount) FILTER (WHERE delegations.party_id != nodes.vega_pub_key), 0) AS staked_by_delegates,
		COALESCE(SUM(delegations.amount), 0) AS staked_total
	FROM nodes
	JOIN delegations ON delegations.node_id = nodes.id
	GROUP BY nodes.id`

	err := pgxscan.Select(ctx, store.pool, &nodes, query)

	return nodes, err
}

func (store *Node) GetNodeByID(ctx context.Context, nodeId string) (entities.Node, error) {
	defer metrics.StartSQLQuery("Node", "GetNodeById")()
	var node entities.Node
	id := entities.NewNodeID(nodeId)

	query := `WITH 
	current_epoch AS (
		SELECT MAX(id) AS id FROM epochs
	),
	delegations AS (
		SELECT * FROM delegations
		WHERE epoch_id = (SELECT id FROM current_epoch)
		AND node_id = $1
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
	
		(CASE WHEN (nodes.reward_score->>'EpochSeq')::BIGINT = (SELECT id FROM current_epoch) THEN nodes.reward_score END) AS reward_score,
		(CASE WHEN (nodes.ranking_score->>'EpochSeq')::BIGINT = (SELECT id FROM current_epoch) THEN nodes.ranking_score END) AS ranking_score,
		
		COALESCE(SUM(delegations.amount) FILTER (WHERE delegations.party_id = nodes.vega_pub_key), 0) AS staked_by_operator,
		COALESCE(SUM(delegations.amount) FILTER (WHERE delegations.party_id != nodes.vega_pub_key), 0) AS staked_by_delegates,
		COALESCE(SUM(delegations.amount), 0) AS staked_total
	FROM nodes
	JOIN delegations ON delegations.node_id = nodes.id
	WHERE nodes.id = $1
	GROUP BY nodes.id`

	err := pgxscan.Get(ctx, store.pool, &node, query, id)
	return node, err
}
