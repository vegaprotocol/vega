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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

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
	query := `INSERT INTO erc20_multisig_signer_events (id, validator_id, signer_change, submitter, nonce, event, vega_time, epoch_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING`

	if _, err := m.pool.Exec(ctx, query,
		e.ID,
		e.ValidatorID,
		e.SignerChange,
		e.Submitter,
		e.Nonce,
		e.Event,
		e.VegaTime,
		e.EpochID,
	); err != nil {
		err = fmt.Errorf("could not insert multisig-signer-events into database: %w", err)
		return err
	}

	return nil
}

func (m *ERC20MultiSigSignerEvent) GetAddedEvents(ctx context.Context, validatorID string, epochID *int64, pagination entities.CursorPagination) (
	[]entities.ERC20MultiSigSignerEvent, entities.PageInfo, error,
) {
	var pageInfo entities.PageInfo
	out := []entities.ERC20MultiSigSignerAddedEvent{}
	sorting, cmp, cursor := extractPaginationInfo(pagination)

	ec := &entities.ERC20MultiSigSignerEventCursor{}
	if err := ec.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to extract pagination information: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, ec.VegaTime),
		NewCursorQueryParameter("id", sorting, cmp, entities.ERC20MultiSigSignerEventID(ec.ID)),
	}

	var args []interface{}
	query := fmt.Sprintf(`SELECT * FROM erc20_multisig_signer_events WHERE validator_id=%s AND event=%s`,
		nextBindVar(&args, entities.NodeID(validatorID)),
		nextBindVar(&args, entities.ERC20MultiSigSignerEventTypeAdded),
	)

	if epochID != nil {
		query = fmt.Sprintf(`%s AND epoch_id=%s`, query, nextBindVar(&args, *epochID))
	}

	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetAddedEvents")()
	if err := pgxscan.Select(ctx, m.pool, &out, query, args...); err != nil {
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
			EpochID:      e.EpochID,
			Event:        e.Event,
		}
	}
	return events, pageInfo, nil
}

func (m *ERC20MultiSigSignerEvent) GetRemovedEvents(ctx context.Context, validatorID string, submitter string, epochID *int64, pagination entities.CursorPagination) ([]entities.ERC20MultiSigSignerEvent, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	out := []entities.ERC20MultiSigSignerRemovedEvent{}
	sorting, cmp, cursor := extractPaginationInfo(pagination)

	ec := &entities.ERC20MultiSigSignerEventCursor{}
	if err := ec.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to extract pagination information: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, ec.VegaTime),
		NewCursorQueryParameter("id", sorting, cmp, entities.ERC20MultiSigSignerEventID(ec.ID)),
	}

	var args []interface{}
	query := fmt.Sprintf(`SELECT * FROM erc20_multisig_signer_events WHERE validator_id=%s AND submitter=%s AND event=%s`,
		nextBindVar(&args, entities.NodeID(validatorID)),
		nextBindVar(&args, entities.EthereumAddress(submitter)),
		nextBindVar(&args, entities.ERC20MultiSigSignerEventTypeRemoved),
	)

	if epochID != nil {
		query = fmt.Sprintf(`%s AND epoch_id=%s`, query, nextBindVar(&args, *epochID))
	}

	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetRemovedEvents")()
	if err := pgxscan.Select(ctx, m.pool, &out, query, args...); err != nil {
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
			VegaTime:     e.VegaTime,
			EpochID:      e.EpochID,
			Event:        e.Event,
		}
	}
	return events, pageInfo, nil
}
