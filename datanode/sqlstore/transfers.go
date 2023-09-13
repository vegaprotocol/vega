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

type Transfers struct {
	*ConnectionSource
}

var transfersOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
}

func NewTransfers(connectionSource *ConnectionSource) *Transfers {
	return &Transfers{
		ConnectionSource: connectionSource,
	}
}

func (t *Transfers) Upsert(ctx context.Context, transfer *entities.Transfer) error {
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
				dispatch_markets,
				reason		
			)
					values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
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
				reason=EXCLUDED.reason,
				tx_hash=EXCLUDED.tx_hash
				;`

	if _, err := t.Connection.Exec(ctx, query, transfer.ID, transfer.TxHash, transfer.VegaTime, transfer.FromAccountID, transfer.ToAccountID,
		transfer.AssetID, transfer.Amount, transfer.Reference, transfer.Status, transfer.TransferType,
		transfer.DeliverOn, transfer.StartEpoch, transfer.EndEpoch, transfer.Factor, transfer.DispatchMetric, transfer.DispatchMetricAsset, transfer.DispatchMarkets, transfer.Reason); err != nil {
		err = fmt.Errorf("could not insert transfer into database: %w", err)
		return err
	}

	return nil
}

func (t *Transfers) GetTransfersToOrFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToOrFromParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)"+
			" or transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to or from party:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *Transfers) GetTransfersFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersFromParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from party:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *Transfers) GetTransfersToParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.Transfer, entities.PageInfo,
	error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to party:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *Transfers) GetTransfersFromAccount(ctx context.Context, accountID entities.AccountID, pagination entities.CursorPagination) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersFromAccount")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination, "WHERE from_account_id = $1", accountID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from account:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *Transfers) GetTransfersToAccount(ctx context.Context, accountID entities.AccountID, pagination entities.CursorPagination) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToAccount")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination, "WHERE to_account_id = $1", accountID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to account:%w", err)
	}

	return transfers, pageInfo, nil
}

func (t *Transfers) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetAll")()
	return t.getTransfers(ctx, pagination, "")
}

func (t *Transfers) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Transfer, error) {
	defer metrics.StartSQLQuery("Transfers", "GetByTxHash")()

	var transfers []entities.Transfer
	query := "SELECT * FROM transfers WHERE tx_hash = $1"

	err := pgxscan.Select(ctx, t.Connection, &transfers, query, txHash)
	if err != nil {
		return nil, fmt.Errorf("getting transfers:%w", err)
	}
	return transfers, nil
}

func (t *Transfers) GetByID(ctx context.Context, id string) (entities.Transfer, error) {
	var t entities.Transfer
	query := `SELECT * FROM transfers_current WHERE id=$1`

	return t, t.wrapE(pgxscan.Get(ctx, ps.Connection, &t, query, entities.TransferID(id)))
}

func (t *Transfers) getTransfers(ctx context.Context, pagination entities.CursorPagination, where string, args ...interface{}) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	var (
		pageInfo entities.PageInfo
		err      error
	)

	query := "select * from transfers_current " + where
	query, args, err = PaginateQuery[entities.TransferCursor](query, args, transfersOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	var transfers []entities.Transfer
	err = pgxscan.Select(ctx, t.Connection, &transfers, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("getting transfers:%w", err)
	}

	transfers, pageInfo = entities.PageEntities[*v2.TransferEdge](transfers, pagination)

	return transfers, pageInfo, nil
}
