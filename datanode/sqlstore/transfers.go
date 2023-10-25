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
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
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
				dispatch_strategy,
				reason		
			)
					values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
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
				dispatch_strategy=EXCLUDED.dispatch_strategy,
				reason=EXCLUDED.reason,
				tx_hash=EXCLUDED.tx_hash
				;`

	if _, err := t.Connection.Exec(ctx, query, transfer.ID, transfer.TxHash, transfer.VegaTime, transfer.FromAccountID, transfer.ToAccountID,
		transfer.AssetID, transfer.Amount, transfer.Reference, transfer.Status, transfer.TransferType,
		transfer.DeliverOn, transfer.StartEpoch, transfer.EndEpoch, transfer.Factor, transfer.DispatchStrategy, transfer.Reason); err != nil {
		err = fmt.Errorf("could not insert transfer into database: %w", err)
		return err
	}

	return nil
}

func (t *Transfers) UpsertFees(ctx context.Context, tf *entities.TransferFees) error {
	defer metrics.StartSQLQuery("Transfers", "UpsertFees")()
	query := `INSERT INTO  transfer_fees(
				transfer_id,
				amount,
				epoch_seq,
				vega_time
			) VALUES ($1, $2, $3, $4) ON CONFLICT (vega_time, transfer_id) DO NOTHING;` // conflicts may occur on checkpoint restore.
	if _, err := t.Connection.Exec(ctx, query, tf.TransferID, tf.Amount, tf.EpochSeq, tf.VegaTime); err != nil {
		return err
	}
	return nil
}

func (t *Transfers) getTransferDetails(ctx context.Context, transfers []entities.Transfer) ([]entities.TransferDetails, error) {
	details := make([]entities.TransferDetails, 0, len(transfers))
	query := `SELECT * FROM transfer_fees WHERE transfer_id = $1`
	for _, tr := range transfers {
		detail := entities.TransferDetails{
			Transfer: tr,
		}
		rows, err := t.Connection.Query(ctx, query, tr.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			details = append(details, detail)
			if rows != nil {
				rows.Close()
			}
			continue
		}
		if err != nil {
			return nil, t.wrapE(err)
		}
		if err := pgxscan.ScanAll(&detail.Fees, rows); err != nil {
			return nil, t.wrapE(err)
		}
		rows.Close()
		details = append(details, detail)
	}
	return details, nil
}

func (t *Transfers) GetTransfersToOrFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.TransferDetails,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToOrFromParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)"+
			" or transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to or from party:%w", err)
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetTransfersFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.TransferDetails,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersFromParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from party:%w", err)
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetTransfersToParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.TransferDetails, entities.PageInfo,
	error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to party:%w", err)
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetTransfersFromAccount(ctx context.Context, accountID entities.AccountID, pagination entities.CursorPagination) ([]entities.TransferDetails,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersFromAccount")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination, "WHERE from_account_id = $1", accountID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from account:%w", err)
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetTransfersToAccount(ctx context.Context, accountID entities.AccountID, pagination entities.CursorPagination) ([]entities.TransferDetails,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToAccount")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination, "WHERE to_account_id = $1", accountID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers to account:%w", err)
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.TransferDetails,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetAll")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination, "")
	if err != nil {
		return nil, entities.PageInfo{}, err
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}
	return details, pageInfo, nil
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

func (t *Transfers) GetByID(ctx context.Context, id string) (entities.TransferDetails, error) {
	var tr entities.Transfer
	query := `SELECT * FROM transfers_current WHERE id=$1`

	if err := pgxscan.Get(ctx, t.Connection, &tr, query, entities.TransferID(id)); err != nil {
		return entities.TransferDetails{}, t.wrapE(err)
	}

	details, err := t.getTransferDetails(ctx, []entities.Transfer{tr})
	if err != nil || len(details) == 0 {
		return entities.TransferDetails{}, err
	}
	return details[0], nil
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

func (t *Transfers) GetAllRewards(ctx context.Context, pagination entities.CursorPagination) ([]entities.Transfer, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetAllRewards")()
	var (
		pageInfo  entities.PageInfo
		err       error
		transfers []entities.Transfer
	)
	query := `SELECT * FROM transfers_current WHERE transfer_type = $1 AND dispatch_strategy->>metric > 0`
	params := []any{entities.Recurring}
	query, params, err = PaginateQuery[entities.TransferCursor](query, params, transfersOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}
	if err = pgxscan.Select(ctx, t.Connection, &transfers, query, params...); err != nil {
		return nil, pageInfo, fmt.Errorf("getting transfers: %w", err)
	}
	transfers, pageInfo = entities.PageEntities[*v2.TransferEdge](transfers, pagination)

	return transfers, pageInfo, nil
}

func (t *Transfers) GetRewardTransfersFromParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.Transfer,
	entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("Transfers", "GetRewardTransfersFromParty")()
	transfers, pageInfo, err := t.getTransfers(ctx, pagination,
		"where transfer_type = $1 AND dispatch_strategy->>metric > 0 AND transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$2)", entities.Recurring, partyID)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("getting transfers from party:%w", err)
	}

	return transfers, pageInfo, nil
}
