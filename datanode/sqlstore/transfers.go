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

type TransferInstructions struct {
	*ConnectionSource
}

var transferInstructionsOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
}

func NewTransferInstructions(connectionSource *ConnectionSource) *TransferInstructions {
	return &TransferInstructions{
		ConnectionSource: connectionSource,
	}
}

func (t *TransferInstructions) Upsert(ctx context.Context, transfer *entities.TransferInstruction) error {
	defer metrics.StartSQLQuery("Transfers", "Upsert")()
	query := `insert into transfers(
				id,
				tx_hash,
				vega_time,
				from_account_id,
				to_account_id,
				asset_id,
				amount,
				reference,
				status,
				transfer_type,
				deliver_on,
				start_epoch,
				end_epoch,
				factor,
				dispatch_metric,
				dispatch_metric_asset,
				dispatch_markets			
			)
					values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
					on conflict (id, vega_time) do update
					set 
				from_account_id=EXCLUDED.from_account_id,
				to_account_id=EXCLUDED.to_account_id,
				asset_id=EXCLUDED.asset_id,
				amount=EXCLUDED.amount,
				reference=EXCLUDED.reference,
				status=EXCLUDED.status,
				transfer_type=EXCLUDED.transfer_type,
				deliver_on=EXCLUDED.deliver_on,
				start_epoch=EXCLUDED.start_epoch,
				end_epoch=EXCLUDED.end_epoch,
				factor=EXCLUDED.factor,
				dispatch_metric=EXCLUDED.dispatch_metric,
				dispatch_metric_asset=EXCLUDED.dispatch_metric_asset,
				dispatch_markets=EXCLUDED.dispatch_markets,
				tx_hash=EXCLUDED.tx_hash
				;`

	if _, err := t.Connection.Exec(ctx, query, transfer.ID, transfer.TxHash, transfer.VegaTime, transfer.FromAccountID, transfer.ToAccountID,
		transfer.AssetID, transfer.Amount, transfer.Reference, transfer.Status, transfer.TransferInstructionType,
		transfer.DeliverOn, transfer.StartEpoch, transfer.EndEpoch, transfer.Factor, transfer.DispatchMetric, transfer.DispatchMetricAsset, transfer.DispatchMarkets); err != nil {
		err = fmt.Errorf("could not insert transfer into database: %w", err)
		return err
	}

	return nil
}

func (t *TransferInstructions) GetTransferInstructionsToOrFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.TransferInstruction,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("TransferInstructions", "GetTransferInstructionsToOrFromParty")()
	transfers, pageInfo, err := t.getTransferInstructions(ctx, pagination,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)"+
			" or transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to or from party:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *TransferInstructions) GetTransferInstructionsFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.TransferInstruction,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("TransferInstructions", "GetTransferInstructionsFromParty")()
	transfers, pageInfo, err := t.getTransferInstructions(ctx, pagination,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from party:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *TransferInstructions) GetTransferInstructionsToParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.TransferInstruction, entities.PageInfo,
	error,
) {
	defer metrics.StartSQLQuery("TransferInstructions", "GetTransferInstructionsToParty")()
	transfers, pageInfo, err := t.getTransferInstructions(ctx, pagination,
		"where transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to party:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *TransferInstructions) GetTransferInstructionsFromAccount(ctx context.Context, accountID int64, pagination entities.CursorPagination) ([]entities.TransferInstruction,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("TransferInstructions", "GetTransferInstructionsFromAccount")()
	transfers, pageInfo, err := t.getTransferInstructions(ctx, pagination, "WHERE from_account_id = $1", accountID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from account:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *TransferInstructions) GetTransferInstructionsToAccount(ctx context.Context, accountID int64, pagination entities.CursorPagination) ([]entities.TransferInstruction,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("TransferInstructions", "GetTransferInstructionsToAccount")()
	transfers, pageInfo, err := t.getTransferInstructions(ctx, pagination, "WHERE to_account_id = $1", accountID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to account:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *TransferInstructions) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.TransferInstruction,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetAll")()
	return t.getTransferInstructions(ctx, pagination, "")
}

func (t *TransferInstructions) getTransferInstructions(ctx context.Context, pagination entities.CursorPagination, where string, args ...interface{}) ([]entities.TransferInstruction,
	entities.PageInfo, error,
) {
	var (
		pageInfo entities.PageInfo
		err      error
	)

	query := "select * from transfers_current " + where
	query, args, err = PaginateQuery[entities.TransferInstructionCursor](query, args, transferInstructionsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	var transfers []entities.TransferInstruction
	err = pgxscan.Select(ctx, t.Connection, &transfers, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("getting transfers:%w", err)
	}

	transfers, pageInfo = entities.PageEntities[*v2.TransferInstructionEdge](transfers, pagination)

	return transfers, pageInfo, nil
}
