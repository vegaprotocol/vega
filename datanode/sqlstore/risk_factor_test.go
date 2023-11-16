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
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskFactors(t *testing.T) {
	t.Run("Make sure you can update risk factors for a market and get latest values", testUpdateMarketRiskFactors)
	t.Run("Upsert should insert risk factor", testAddRiskFactor)
	t.Run("Upsert should update the risk factor if the market already exists in the same block", testUpsertDuplicateMarketInSameBlock)
	t.Run("GetMarketRiskFactors returns the risk factors for the given market id", testGetMarketRiskFactors)
}

func setupRiskFactorTests(t *testing.T) (*sqlstore.Blocks, *sqlstore.RiskFactors) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	rfStore := sqlstore.NewRiskFactors(connectionSource)
	return bs, rfStore
}

func testUpdateMarketRiskFactors(t *testing.T) {
	ctx := tempTransaction(t)

	bs, rfStore := setupRiskFactorTests(t)

	// Add a risk factor for market 'aa' in one block

	source := &testBlockSource{bs, time.Now()}
	block := source.getNextBlock(t, ctx)
	marketID := entities.MarketID("aa")
	rf := entities.RiskFactor{
		MarketID: marketID,
		Short:    decimal.NewFromInt(100),
		Long:     decimal.NewFromInt(200),
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
	}
	rfStore.Upsert(ctx, &rf)

	// Check we get the same data back as we put in
	fetched, err := rfStore.GetMarketRiskFactors(ctx, string(marketID))
	require.NoError(t, err)
	assert.Equal(t, fetched, rf)

	// Upsert a new risk factor for the same in a different block
	block2 := source.getNextBlock(t, ctx)
	rf2 := entities.RiskFactor{
		MarketID: marketID,
		Short:    decimal.NewFromInt(101),
		Long:     decimal.NewFromInt(202),
		TxHash:   generateTxHash(),
		VegaTime: block2.VegaTime,
	}
	rfStore.Upsert(ctx, &rf2)

	// Check we get back the updated version
	fetched, err = rfStore.GetMarketRiskFactors(ctx, string(marketID))
	require.NoError(t, err)
	assert.Equal(t, fetched, rf2)
}

func testAddRiskFactor(t *testing.T) {
	ctx := tempTransaction(t)

	bs, rfStore := setupRiskFactorTests(t)

	var rowCount int
	err := connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, ctx, bs)
	riskFactorProto := getRiskFactorProto()
	riskFactor, err := entities.RiskFactorFromProto(riskFactorProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting risk factor proto to database entity")

	err = rfStore.Upsert(ctx, riskFactor)
	require.NoError(t, err)

	err = connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testUpsertDuplicateMarketInSameBlock(t *testing.T) {
	ctx := tempTransaction(t)

	bs, rfStore := setupRiskFactorTests(t)

	var rowCount int
	err := connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, ctx, bs)
	riskFactorProto := getRiskFactorProto()
	riskFactor, err := entities.RiskFactorFromProto(riskFactorProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting risk factor proto to database entity")

	err = rfStore.Upsert(ctx, riskFactor)
	require.NoError(t, err)

	err = connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	err = rfStore.Upsert(ctx, riskFactor)
	require.NoError(t, err)

	err = connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func getRiskFactorProto() *vega.RiskFactor {
	return &vega.RiskFactor{
		Market: "deadbeef",
		Short:  "1000",
		Long:   "1000",
	}
}

func testGetMarketRiskFactors(t *testing.T) {
	ctx := tempTransaction(t)

	bs, rfStore := setupRiskFactorTests(t)

	var rowCount int
	err := connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, ctx, bs)
	riskFactorProto := getRiskFactorProto()
	riskFactor, err := entities.RiskFactorFromProto(riskFactorProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting risk factor proto to database entity")

	err = rfStore.Upsert(ctx, riskFactor)
	require.NoError(t, err)

	err = connectionSource.Connection.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	got, err := rfStore.GetMarketRiskFactors(ctx, "DEADBEEF")
	assert.NoError(t, err)

	want := *riskFactor

	assert.Equal(t, want, got)
}
