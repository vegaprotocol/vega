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

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/require"
)

type TransferOption func(*testing.T, *eventspb.Transfer)

func CreateTransfer(t *testing.T, ctx context.Context, transferStore *sqlstore.Transfers, accountsStore *sqlstore.Accounts, block entities.Block, options ...TransferOption) *entities.Transfer {
	t.Helper()

	transfer := NewTransfer(t, ctx, accountsStore, block, options...)

	require.NoError(t, transferStore.Upsert(ctx, transfer))

	return transfer
}

func NewTransfer(t *testing.T, ctx context.Context, accountsStore *sqlstore.Accounts, block entities.Block, options ...TransferOption) *entities.Transfer {
	t.Helper()

	// Postgres only stores timestamps in microsecond resolution.
	// Without truncating, the timestamp will mismatch in test assertions.
	blockTimeMs := block.VegaTime.Truncate(time.Microsecond)

	transferEvent := &eventspb.Transfer{
		Id:              GenerateID(),
		From:            GenerateID(),
		FromAccountType: vegapb.AccountType_ACCOUNT_TYPE_GENERAL,
		To:              GenerateID(),
		ToAccountType:   vegapb.AccountType_ACCOUNT_TYPE_GENERAL,
		Asset:           GenerateID(),
		Amount:          vgtest.RandomPositiveU64AsString(),
		Reference:       GenerateID(),
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       blockTimeMs.UnixNano(),
	}

	for _, option := range options {
		option(t, transferEvent)
	}

	if transferEvent.Kind == nil {
		t.Fatal("transfer is missing a kind")
	}

	transfer, err := entities.TransferFromProto(ctx, transferEvent, generateTxHash(), blockTimeMs, accountsStore)
	require.NoError(t, err)

	return transfer
}

func TransferWithID(id entities.TransferID) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.Id = id.String()
	}
}

func TransferWithStatus(status entities.TransferStatus) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.Status = eventspb.Transfer_Status(status)
	}
}

func TransferWithAsset(asset *entities.Asset) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.Asset = asset.ID.String()
	}
}

func TransferFromToAccounts(from, to *entities.Account) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.From = from.PartyID.String()
		transfer.FromAccountType = from.Type
		transfer.To = to.PartyID.String()
		transfer.ToAccountType = to.Type
	}
}

func TransferAsRecurring(config eventspb.RecurringTransfer) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.Kind = &eventspb.Transfer_Recurring{
			Recurring: &config,
		}
	}
}

func TransferAsRecurringGovernance(config eventspb.RecurringGovernanceTransfer) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.Kind = &eventspb.Transfer_RecurringGovernance{
			RecurringGovernance: &config,
		}
	}
}

func TransferAsOneOff(config eventspb.OneOffTransfer) TransferOption {
	return func(t *testing.T, transfer *eventspb.Transfer) {
		t.Helper()
		transfer.Kind = &eventspb.Transfer_OneOff{
			OneOff: &config,
		}
	}
}

func TransferDetailsAsTransfers(t *testing.T, details []entities.TransferDetails) []entities.Transfer {
	t.Helper()

	transfers := make([]entities.Transfer, 0, len(details))
	for i := range details {
		transfers = append(transfers, details[i].Transfer)
	}
	return transfers
}
