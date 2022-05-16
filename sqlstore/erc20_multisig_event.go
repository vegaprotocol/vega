package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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

func (m *ERC20MultiSigSignerEvent) GetByValidatorID(ctx context.Context, validatorID string, submitter string, eventType entities.ERC20MultiSigSignerEventType, epochID *int64, pagination entities.OffsetPagination) ([]entities.ERC20MultiSigSignerEvent, error) {
	out := []entities.ERC20MultiSigSignerEvent{}
	prequery := `SELECT * FROM erc20_multisig_signer_events WHERE validator_id=$1`
	query, args := orderAndPaginateQuery(prequery, nil, pagination, entities.NewNodeID(validatorID))

	if epochID != nil {
		prequery += " AND epoch_id=$2"
		query, args = orderAndPaginateQuery(prequery, nil, pagination, entities.NewNodeID(validatorID), *epochID)
	}
	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetByValidatorID")()
	err := pgxscan.Select(ctx, m.pool, &out, query, args...)
	return out, err
}

func (m *ERC20MultiSigSignerEvent) GetAddedEvents(ctx context.Context, validatorID string, epochID *int64, pagination entities.OffsetPagination) ([]entities.ERC20MultiSigSignerEvent, error) {
	out := []entities.ERC20MultiSigSignerEvent{}
	prequery := `SELECT * FROM erc20_multisig_signer_events WHERE validator_id=$1 AND event=$2`
	query, args := orderAndPaginateQuery(prequery, nil, pagination, entities.NewNodeID(validatorID), entities.ERC20MultiSigSignerEventTypeAdded)

	if epochID != nil {
		prequery += " AND epoch_id=$3"
		query, args = orderAndPaginateQuery(prequery, nil, pagination, entities.NewNodeID(validatorID), entities.ERC20MultiSigSignerEventTypeAdded, *epochID)
	}

	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetAddedEvents")()
	err := pgxscan.Select(ctx, m.pool, &out, query, args...)
	return out, err
}

func (m *ERC20MultiSigSignerEvent) GetRemovedEvents(ctx context.Context, validatorID string, submitter string, epochID *int64, pagination entities.OffsetPagination) ([]entities.ERC20MultiSigSignerEvent, error) {
	out := []entities.ERC20MultiSigSignerEvent{}
	prequery := `SELECT * FROM erc20_multisig_signer_events WHERE validator_id=$1 AND submitter=$2 AND event=$3`
	query, args := orderAndPaginateQuery(prequery, nil, pagination, entities.NewNodeID(validatorID), entities.NewEthereumAddress(submitter), entities.ERC20MultiSigSignerEventTypeRemoved)

	if epochID != nil {
		prequery += " AND epoch_id=$4"
		query, args = orderAndPaginateQuery(prequery, nil, pagination, entities.NewNodeID(validatorID), entities.NewEthereumAddress(submitter), entities.ERC20MultiSigSignerEventTypeRemoved, *epochID)
	}

	defer metrics.StartSQLQuery("ERC20MultiSigSignerEvent", "GetRemovedEvents")()
	err := pgxscan.Select(ctx, m.pool, &out, query, args...)
	return out, err
}
