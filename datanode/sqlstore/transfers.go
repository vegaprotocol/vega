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

package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

var transfersOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
}

type Transfers struct {
	*ConnectionSource
}

type ListTransfersFilters struct {
	FromEpoch *uint64
	ToEpoch   *uint64
	Scope     *entities.TransferScope
	Status    *entities.TransferStatus
	GameID    *entities.GameID
}

func NewTransfers(connectionSource *ConnectionSource) *Transfers {
	return &Transfers{
		ConnectionSource: connectionSource,
	}
}

func (t *Transfers) Upsert(ctx context.Context, transfer *entities.Transfer) error {
	defer metrics.StartSQLQuery("Transfers", "Upsert")()
	query := `INSERT INTO transfers(
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
				reason,
				game_id
			)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
					ON CONFLICT (id, vega_time) DO UPDATE
					SET
				from_account_id=excluded.from_account_id,
				to_account_id=excluded.to_account_id,
				asset_id=excluded.asset_id,
				amount=excluded.amount,
				reference=excluded.reference,
				status=excluded.status,
				transfer_type=excluded.transfer_type,
				deliver_on=excluded.deliver_on,
				start_epoch=excluded.start_epoch,
				end_epoch=excluded.end_epoch,
				factor=excluded.factor,
				dispatch_strategy=excluded.dispatch_strategy,
				reason=excluded.reason,
				tx_hash=excluded.tx_hash,
				game_id=excluded.game_id
				;`

	if _, err := t.Connection.Exec(ctx, query, transfer.ID, transfer.TxHash, transfer.VegaTime, transfer.FromAccountID, transfer.ToAccountID,
		transfer.AssetID, transfer.Amount, transfer.Reference, transfer.Status, transfer.TransferType,
		transfer.DeliverOn, transfer.StartEpoch, transfer.EndEpoch, transfer.Factor, transfer.DispatchStrategy, transfer.Reason, transfer.GameID); err != nil {
		return fmt.Errorf("could not insert transfer into database: %w", err)
	}

	return nil
}

func (t *Transfers) UpsertFees(ctx context.Context, tf *entities.TransferFees) error {
	defer metrics.StartSQLQuery("Transfers", "UpsertFees")()
	query := `INSERT INTO  transfer_fees(
				transfer_id,
				amount,
				epoch_seq,
				vega_time,
				discount_applied
			) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (vega_time, transfer_id) DO NOTHING;` // conflicts may occur on checkpoint restore.
	if _, err := t.Connection.Exec(ctx, query, tf.TransferID, tf.Amount, tf.EpochSeq, tf.VegaTime, tf.DiscountApplied); err != nil {
		return err
	}
	return nil
}

func (t *Transfers) GetTransfersToOrFromParty(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters, partyID entities.PartyID) ([]entities.TransferDetails, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToOrFromParty")()

	where := []string{
		"(transfers_current.from_account_id in (select id from accounts where accounts.party_id=$1) or transfers_current.to_account_id in (select id from accounts where accounts.party_id=$1))",
	}

	transfers, pageInfo, err := t.getCurrentTransfers(ctx, pagination, filters, where, []any{partyID})
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("could not get transfers to or from party: %w", err)
	}

	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetTransfersFromParty(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters, partyID entities.PartyID) ([]entities.TransferDetails, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersFromParty")()

	where := []string{
		"transfers_current.from_account_id in (select id from accounts where accounts.party_id=$1)",
	}

	transfers, pageInfo, err := t.getCurrentTransfers(ctx, pagination, filters, where, []any{partyID})
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("could not get transfers from party: %w", err)
	}
	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetTransfersToParty(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters, partyID entities.PartyID) ([]entities.TransferDetails, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetTransfersToParty")()

	where := []string{
		"transfers_current.to_account_id in (select id from accounts where accounts.party_id=$1)",
	}

	transfers, pageInfo, err := t.getCurrentTransfers(ctx, pagination, filters, where, []any{partyID})
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("could not get transfers to party: %w", err)
	}

	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetAll(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters) ([]entities.TransferDetails, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetAll")()

	transfers, pageInfo, err := t.getCurrentTransfers(ctx, pagination, filters, nil, nil)
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
	query := "SELECT * FROM transfers WHERE tx_hash = $1 ORDER BY id"

	if err := pgxscan.Select(ctx, t.Connection, &transfers, query, txHash); err != nil {
		return nil, fmt.Errorf("could not get transfers by transaction hash: %w", err)
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

func (t *Transfers) GetAllRewards(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters) ([]entities.TransferDetails, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetAllRewards")()

	args := []any{entities.Recurring, entities.GovernanceRecurring}

	transfers, pageInfo, err := t.getRecurringTransfers(ctx, pagination, filters, []string{}, args)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("could not get recurring transfers: %w", err)
	}

	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) GetRewardTransfersFromParty(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters, partyID entities.PartyID) ([]entities.TransferDetails, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Transfers", "GetRewardTransfersFromParty")()

	where := []string{
		"from_account_id IN (SELECT id FROM accounts WHERE accounts.party_id = $3)",
	}

	args := []any{entities.Recurring, entities.GovernanceRecurring, partyID}

	transfers, pageInfo, err := t.getRecurringTransfers(ctx, pagination, filters, where, args)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("could not get recurring transfers: %w", err)
	}

	details, err := t.getTransferDetails(ctx, transfers)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	return details, pageInfo, nil
}

func (t *Transfers) UpsertFeesDiscount(ctx context.Context, tfd *entities.TransferFeesDiscount) error {
	defer metrics.StartSQLQuery("Transfers", "UpsertFeesDiscount")()
	query := `INSERT INTO transfer_fees_discount(
				party_id,
				asset_id,
				amount,
				epoch_seq,
				vega_time
			) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (vega_time, party_id, asset_id) DO NOTHING ;` // conflicts may occur on checkpoint restore.
	if _, err := t.Connection.Exec(ctx, query, tfd.PartyID, tfd.AssetID, tfd.Amount, tfd.EpochSeq, tfd.VegaTime); err != nil {
		return err
	}
	return nil
}

func (t *Transfers) GetCurrentTransferFeeDiscount(
	ctx context.Context,
	partyID entities.PartyID,
	assetID entities.AssetID,
) (*entities.TransferFeesDiscount, error) {
	defer metrics.StartSQLQuery("Transfers", "GetCurrentTransferFeeDiscount")()

	var tfd entities.TransferFeesDiscount
	query := `SELECT * FROM transfer_fees_discount
		WHERE party_id = $1 AND asset_id = $2
		ORDER BY vega_time DESC LIMIT 1`

	if err := pgxscan.Get(ctx, t.Connection, &tfd, query, partyID, assetID); err != nil {
		return &entities.TransferFeesDiscount{}, t.wrapE(err)
	}

	return &tfd, nil
}

func (t *Transfers) getCurrentTransfers(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters, where []string, args []any) ([]entities.Transfer, entities.PageInfo, error) {
	whereStr, args := t.buildWhereClause(filters, where, args)
	query := "SELECT * FROM transfers_current " + whereStr

	return t.selectTransfers(ctx, pagination, query, args)
}

func (t *Transfers) getRecurringTransfers(ctx context.Context, pagination entities.CursorPagination, filters ListTransfersFilters, where []string, args []any) ([]entities.Transfer, entities.PageInfo, error) {
	whereStr, args := t.buildWhereClause(filters, where, args)

	query := `WITH recurring_transfers AS (
		SELECT tc.* FROM transfers_current as tc
		JOIN accounts as a on tc.to_account_id = a.id
		WHERE transfer_type IN ($1, $2)
		AND a.type = 12 OR (jsonb_typeof(tc.dispatch_strategy) != 'null' AND dispatch_strategy->>'metric' <> '0')
)
SELECT *
FROM recurring_transfers
` + whereStr

	return t.selectTransfers(ctx, pagination, query, args)
}

func (t *Transfers) buildWhereClause(filters ListTransfersFilters, where []string, args []any) (string, []any) {
	if filters.Scope != nil {
		where = append(where, "jsonb_typeof(dispatch_strategy) != 'null'")
		switch *filters.Scope {
		case entities.TransferScopeIndividual:
			where = append(where, "dispatch_strategy ? 'individual_scope'")
		case entities.TransferScopeTeam:
			where = append(where, "dispatch_strategy ? 'team_scope'")
		}
	}

	if filters.Status != nil {
		where = append(where, fmt.Sprintf("status = %s", nextBindVar(&args, *filters.Status)))
	}

	if filters.FromEpoch != nil {
		where = append(where, fmt.Sprintf("(start_epoch >= %s or end_epoch >= %s)",
			nextBindVar(&args, *filters.FromEpoch),
			nextBindVar(&args, *filters.FromEpoch),
		))
	}

	if filters.ToEpoch != nil {
		where = append(where, fmt.Sprintf("(start_epoch <= %s or end_epoch <= %s)",
			nextBindVar(&args, *filters.ToEpoch),
			nextBindVar(&args, *filters.ToEpoch),
		))
	}

	if filters.GameID != nil {
		where = append(where, fmt.Sprintf("game_id = %s", nextBindVar(&args, *filters.GameID)))
	}

	whereStr := ""
	if len(where) > 0 {
		whereStr = "where " + strings.Join(where, " and ")
	}
	return whereStr, args
}

func (t *Transfers) selectTransfers(ctx context.Context, pagination entities.CursorPagination, query string, args []any) ([]entities.Transfer, entities.PageInfo, error) {
	query, args, err := PaginateQuery[entities.TransferCursor](query, args, transfersOrdering, pagination)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	var transfers []entities.Transfer
	err = pgxscan.Select(ctx, t.Connection, &transfers, query, args...)
	if err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("could not get transfers: %w", err)
	}

	transfers, pageInfo := entities.PageEntities[*v2.TransferEdge](transfers, pagination)

	return transfers, pageInfo, nil
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
