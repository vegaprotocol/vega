package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Transfers struct {
	*ConnectionSource
}

func NewTransfers(connectionSource *ConnectionSource) *Transfers {
	return &Transfers{
		ConnectionSource: connectionSource,
	}
}

func (t *Transfers) Upsert(ctx context.Context, transfer *entities.Transfer) error {

	query := `insert into transfers(
				id,
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
				end_epoch,factor)
					values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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
				factor=EXCLUDED.factor
				;`

	if _, err := t.Connection.Exec(ctx, query, transfer.ID, transfer.VegaTime, transfer.FromAccountId, transfer.ToAccountId,
		transfer.AssetId, transfer.Amount, transfer.Reference, transfer.Status, transfer.TransferType,
		transfer.DeliverOn, transfer.StartEpoch, transfer.EndEpoch, transfer.Factor); err != nil {
		err = fmt.Errorf("could not insert transfer into database: %w", err)
		return err
	}

	return nil
}

func (t *Transfers) GetTransfersFromParty(ctx context.Context, partyId entities.PartyID) ([]*entities.Transfer, error) {
	transfers, err := t.getTransfers(ctx,
		"where transfers_current.from_account_id  in (select id from accounts where accounts.party_id=$1)", partyId)

	if err != nil {
		return nil, fmt.Errorf("getting transfers from party:%w", err)
	}

	return transfers, nil
}

func (t *Transfers) GetTransfersToParty(ctx context.Context, partyId entities.PartyID) ([]*entities.Transfer, error) {
	transfers, err := t.getTransfers(ctx,
		"where transfers_current.to_account_id  in (select id from accounts where accounts.party_id=$1)", partyId)

	if err != nil {
		return nil, fmt.Errorf("getting transfers to party:%w", err)
	}

	return transfers, nil
}

func (t *Transfers) GetTransfersFromAccount(ctx context.Context, accountID int64) ([]*entities.Transfer, error) {
	transfers, err := t.getTransfers(ctx, "WHERE from_account_id = $1", accountID)

	if err != nil {
		return nil, fmt.Errorf("getting transfers from account:%w", err)
	}

	return transfers, nil
}

func (t *Transfers) GetTransfersToAccount(ctx context.Context, accountID int64) ([]*entities.Transfer, error) {
	transfers, err := t.getTransfers(ctx, "WHERE to_account_id = $1", accountID)

	if err != nil {
		return nil, fmt.Errorf("getting transfers to account:%w", err)
	}

	return transfers, nil
}

func (t *Transfers) GetAll(ctx context.Context) ([]*entities.Transfer, error) {
	return t.getTransfers(ctx, "")
}

func (t *Transfers) getTransfers(ctx context.Context, where string, args ...interface{}) ([]*entities.Transfer, error) {
	var transfers []*entities.Transfer
	query := "select * from transfers_current " + where
	err := pgxscan.Select(ctx, t.Connection, &transfers, query, args...)
	if err != nil {
		return nil, fmt.Errorf("getting transfers:%w", err)
	}

	return transfers, nil
}
