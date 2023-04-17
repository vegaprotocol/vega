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

var erc20MultSigSignerOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
}

type ERC20MultiSigSignerEvent struct {
	*ConnectionSource
}

func NewERC20MultiSigSignerEvent(connectionSource *ConnectionSource) *ERC20MultiSigSignerEvent {
	return &ERC20MultiSigSignerEvent{
		ConnectionSource: connectionSource,
	}
}

func (m *ERC20MultiSigSignerEvent) Add(ctx context.Context, e *entities.ERC20MultiSigSignerEvent) error {
	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "Add")()
	query := `INSERT INTO erc20_multisig_signer_events (id, validator_id, signer_change, submitter, nonce, event, tx_hash, vega_time, epoch_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING`

	if _, err := m.Connection.Exec(ctx, query,
		e.ID,
		e.ValidatorID,
		e.SignerChange,
		e.Submitter,
		e.Nonce,
		e.Event,
		e.TxHash,
		e.VegaTime,
		e.EpochID,
	); err != nil {
		err = fmt.Errorf("could not insert multisig-signer-events into database: %w", err)
		return err
	}

	return nil
}

func (m *ERC20MultiSigSignerEvent) GetAddedEvents(ctx context.Context, validatorID string, submitter string, epochID *int64, pagination entities.CursorPagination) (
	[]entities.ERC20MultiSigSignerEvent, entities.PageInfo, error,
) {
	var pageInfo entities.PageInfo
	out := []entities.ERC20MultiSigSignerAddedEvent{}

	var args []interface{}

	conditions := []string{}
	if validatorID != "" {
		conditions = append(conditions, fmt.Sprintf("validator_id=%s", nextBindVar(&args, entities.NodeID(validatorID))))
	}

	if submitter != "" {
		conditions = append(conditions, fmt.Sprintf("submitter=%s", nextBindVar(&args, entities.EthereumAddress(submitter))))
	}

	if epochID != nil {
		conditions = append(conditions, fmt.Sprintf("epoch_id=%s", nextBindVar(&args, *epochID)))
	}

	conditions = append(conditions, fmt.Sprintf("event=%s", nextBindVar(&args, entities.ERC20MultiSigSignerEventTypeAdded)))

	query := `SELECT * FROM erc20_multisig_signer_events`
	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	query, args, err := PaginateQuery[entities.ERC20MultiSigSignerEventCursor](query, args, erc20MultSigSignerOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetAddedEvents")()
	if err = pgxscan.Select(ctx, m.Connection, &out, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to retrieve multisig signer events: %w", err)
	}

	out, pageInfo = entities.PageEntities[*v2.ERC20MultiSigSignerAddedEdge](out, pagination)

	events := make([]entities.ERC20MultiSigSignerEvent, len(out))
	for i, e := range out {
		events[i] = entities.ERC20MultiSigSignerEvent{
			ID:           e.ID,
			ValidatorID:  e.ValidatorID,
			SignerChange: e.SignerChange,
			Submitter:    e.Submitter,
			Nonce:        e.Nonce,
			VegaTime:     e.VegaTime,
			TxHash:       e.TxHash,
			EpochID:      e.EpochID,
			Event:        e.Event,
		}
	}
	return events, pageInfo, nil
}

func (m *ERC20MultiSigSignerEvent) GetRemovedEvents(ctx context.Context, validatorID string, submitter string, epochID *int64, pagination entities.CursorPagination) ([]entities.ERC20MultiSigSignerEvent, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	var err error
	out := []entities.ERC20MultiSigSignerRemovedEvent{}
	var args []interface{}

	conditions := []string{}
	if validatorID != "" {
		conditions = append(conditions, fmt.Sprintf("validator_id=%s", nextBindVar(&args, entities.NodeID(validatorID))))
	}

	if submitter != "" {
		conditions = append(conditions, fmt.Sprintf("submitter=%s", nextBindVar(&args, entities.EthereumAddress(submitter))))
	}

	if epochID != nil {
		conditions = append(conditions, fmt.Sprintf("epoch_id=%s", nextBindVar(&args, *epochID)))
	}

	conditions = append(conditions, fmt.Sprintf("event=%s", nextBindVar(&args, entities.ERC20MultiSigSignerEventTypeRemoved)))

	query := `SELECT * FROM erc20_multisig_signer_events`
	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	query, args, err = PaginateQuery[entities.ERC20MultiSigSignerEventCursor](query, args, erc20MultSigSignerOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetRemovedEvents")()
	if err = pgxscan.Select(ctx, m.Connection, &out, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to retrieve multisig signer events: %w", err)
	}

	out, pageInfo = entities.PageEntities[*v2.ERC20MultiSigSignerRemovedEdge](out, pagination)

	events := make([]entities.ERC20MultiSigSignerEvent, len(out))
	for i, e := range out {
		events[i] = entities.ERC20MultiSigSignerEvent{
			ID:           e.ID,
			ValidatorID:  e.ValidatorID,
			SignerChange: e.SignerChange,
			Submitter:    e.Submitter,
			Nonce:        e.Nonce,
			TxHash:       e.TxHash,
			VegaTime:     e.VegaTime,
			EpochID:      e.EpochID,
			Event:        e.Event,
		}
	}
	return events, pageInfo, nil
}

func (m *ERC20MultiSigSignerEvent) GetRemovedByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.ERC20MultiSigSignerRemovedEvent, error) {
	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetRemovedByTxHash")()

	var events []entities.ERC20MultiSigSignerRemovedEvent
	query := `SELECT * FROM erc20_multisig_signer_events WHERE event=$1 AND tx_hash = $2`

	if err := pgxscan.Select(ctx, m.Connection, &events, query, entities.ERC20MultiSigSignerEventTypeRemoved, txHash); err != nil {
		return nil, fmt.Errorf("failed to retrieve multisig removed signer events: %w", err)
	}

	return events, nil
}

func (m *ERC20MultiSigSignerEvent) GetAddedByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.ERC20MultiSigSignerAddedEvent, error) {
	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetAddedByTxHash")()

	var events []entities.ERC20MultiSigSignerAddedEvent
	query := `SELECT * FROM erc20_multisig_signer_events WHERE event=$1 AND tx_hash = $2`

	if err := pgxscan.Select(ctx, m.Connection, &events, query, entities.ERC20MultiSigSignerEventTypeAdded, txHash); err != nil {
		return nil, fmt.Errorf("failed to retrieve multisig added signer events: %w", err)
	}

	return events, nil
}
