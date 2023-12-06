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

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func CreateAsset(t *testing.T, ctx context.Context, store *sqlstore.Assets, block entities.Block) *entities.Asset {
	t.Helper()

	asset := NewAsset(t, block)

	require.NoError(t, store.Add(ctx, *asset))

	return asset
}

func NewAsset(t *testing.T, block entities.Block) *entities.Asset {
	t.Helper()

	// Postgres only stores timestamps in microsecond resolution.
	// Without truncating, the timestamp will mismatch in test assertions.
	blockTimeMs := block.VegaTime.Truncate(time.Microsecond)

	asset := &entities.Asset{
		ID:            entities.AssetID(GenerateID()),
		Name:          "TestAssetName",
		Symbol:        "TAN",
		Decimals:      1,
		Quantum:       decimal.NewFromInt(1),
		Source:        "TS",
		ERC20Contract: "ET",
		VegaTime:      blockTimeMs,
	}

	return asset
}
