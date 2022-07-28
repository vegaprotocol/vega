// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskFactors(t *testing.T) {
	t.Run("Upsert should insert risk factor", testAddRiskFactor)
	t.Run("Upsert should update the risk factor if the market already exists in the same block", testUpsertDuplicateMarketInSameBlock)
	t.Run("GetMarketRiskFactors returns the risk factors for the given market id", testGetMarketRiskFactors)
}

func setupRiskFactorTests(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.RiskFactors, *pgx.Conn) {
	t.Helper()
	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	rfStore := sqlstore.NewRiskFactors(connectionSource)
	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return bs, rfStore, conn
}

func testAddRiskFactor(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, rfStore, conn := setupRiskFactorTests(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, bs)
	riskFactorProto := getRiskFactorProto()
	riskFactor, err := entities.RiskFactorFromProto(riskFactorProto, block.VegaTime)
	require.NoError(t, err, "Converting risk factor proto to database entity")

	err = rfStore.Upsert(context.Background(), riskFactor)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testUpsertDuplicateMarketInSameBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, rfStore, conn := setupRiskFactorTests(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, bs)
	riskFactorProto := getRiskFactorProto()
	riskFactor, err := entities.RiskFactorFromProto(riskFactorProto, block.VegaTime)
	require.NoError(t, err, "Converting risk factor proto to database entity")

	err = rfStore.Upsert(context.Background(), riskFactor)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	err = rfStore.Upsert(context.Background(), riskFactor)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
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
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, rfStore, conn := setupRiskFactorTests(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, bs)
	riskFactorProto := getRiskFactorProto()
	riskFactor, err := entities.RiskFactorFromProto(riskFactorProto, block.VegaTime)
	require.NoError(t, err, "Converting risk factor proto to database entity")

	err = rfStore.Upsert(context.Background(), riskFactor)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from risk_factors`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	got, err := rfStore.GetMarketRiskFactors(ctx, "DEADBEEF")
	assert.NoError(t, err)

	want := *riskFactor

	assert.Equal(t, want, got)
}
