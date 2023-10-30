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

package helpers

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/stretchr/testify/require"
)

func AddTestAccount(t *testing.T,
	ctx context.Context,
	accountStore *sqlstore.Accounts,
	party entities.Party,
	asset entities.Asset,
	accountType types.AccountType,
	block entities.Block,
) entities.Account {
	t.Helper()
	account := entities.Account{
		PartyID:  party.ID,
		AssetID:  asset.ID,
		MarketID: entities.MarketID(GenerateID()),
		Type:     accountType,
		VegaTime: block.VegaTime,
	}

	err := accountStore.Add(ctx, &account)
	require.NoError(t, err)
	return account
}

func AddTestAccountWithTxHash(t *testing.T,
	ctx context.Context,
	accountStore *sqlstore.Accounts,
	party entities.Party,
	asset entities.Asset,
	accountType types.AccountType,
	block entities.Block,
	txHash entities.TxHash,
) entities.Account {
	t.Helper()
	account := entities.Account{
		PartyID:  party.ID,
		AssetID:  asset.ID,
		MarketID: entities.MarketID(GenerateID()),
		Type:     accountType,
		VegaTime: block.VegaTime,
		TxHash:   txHash,
	}

	err := accountStore.Add(ctx, &account)
	require.NoError(t, err)
	return account
}

func AddTestAccountWithMarketAndType(t *testing.T,
	ctx context.Context,
	accountStore *sqlstore.Accounts,
	party entities.Party,
	asset entities.Asset,
	block entities.Block,
	market entities.MarketID,
	accountType types.AccountType,
) entities.Account {
	t.Helper()
	account := entities.Account{
		PartyID:  party.ID,
		AssetID:  asset.ID,
		MarketID: market,
		Type:     accountType,
		VegaTime: block.VegaTime,
	}

	err := accountStore.Add(ctx, &account)
	require.NoError(t, err)
	return account
}
