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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

type AccountOption func(*testing.T, *entities.Account)

func CreateAccount(t *testing.T, ctx context.Context, store *sqlstore.Accounts, block entities.Block, options ...AccountOption) *entities.Account {
	t.Helper()

	account := NewAccount(t, block, options...)

	require.NoError(t, store.Obtain(ctx, account))

	return account
}

func NewAccount(t *testing.T, block entities.Block, options ...AccountOption) *entities.Account {
	t.Helper()

	// Postgres only stores timestamps in microsecond resolution.
	// Without truncating, the timestamp will mismatch in test assertions.
	blockTimeMs := block.VegaTime.Truncate(time.Microsecond)

	account := &entities.Account{
		ID:       entities.AccountID(GenerateID()),
		PartyID:  entities.PartyID(GenerateID()),
		AssetID:  entities.AssetID(GenerateID()),
		Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
		VegaTime: blockTimeMs,
	}

	for _, option := range options {
		option(t, account)
	}

	return account
}

func AccountForAsset(asset *entities.Asset) AccountOption {
	return func(t *testing.T, account *entities.Account) {
		t.Helper()
		account.AssetID = asset.ID
	}
}

func AccountWithType(accountType vega.AccountType) AccountOption {
	return func(t *testing.T, account *entities.Account) {
		t.Helper()
		account.Type = accountType
	}
}
