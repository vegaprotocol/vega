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

package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"

	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestOracleSpec(t *testing.T) {
	t.Run("Upsert should insert an OracleSpec when the id does not exist in the current block", testInsertIntoNewBlock)
	t.Run("Upsert should update an OracleSpec when the id already exists in the current block", testUpdateExistingInBlock)
	t.Run("GetSpecByID should retrieve the latest version of the specified OracleSpec", testGetSpecByID)
	t.Run("GetByTxHash", testGetSpecByTxHash)
}

func setupOracleSpecTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.OracleSpec, sqlstore.Connection) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	os := sqlstore.NewOracleSpec(connectionSource)

	return bs, os, connectionSource.Connection
}

func testInsertIntoNewBlock(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, conn := setupOracleSpecTest(t)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	specProtos := getTestSpecs()

	proto := specProtos[0]
	data := entities.OracleSpecFromProto(proto, generateTxHash(), block.VegaTime)
	assert.NoError(t, os.Upsert(ctx, data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)

	proto = specProtos[4]
	data = entities.OracleSpecFromProto(proto, generateTxHash(), block.VegaTime)
	assert.NoError(t, os.Upsert(ctx, data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 2, rowCount)
}

func testUpdateExistingInBlock(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, conn := setupOracleSpecTest(t)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	specProtos := getTestSpecs()

	proto := specProtos[0]
	data := entities.OracleSpecFromProto(proto, generateTxHash(), block.VegaTime)
	assert.NoError(t, os.Upsert(ctx, data))

	data.ExternalDataSourceSpec.Spec.Status = entities.OracleSpecDeactivated
	assert.NoError(t, os.Upsert(ctx, data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)

	proto = specProtos[4]
	data = entities.OracleSpecFromProto(proto, generateTxHash(), block.VegaTime)
	assert.NoError(t, os.Upsert(ctx, data))

	data.ExternalDataSourceSpec.Spec.Status = entities.OracleSpecDeactivated
	assert.NoError(t, os.Upsert(ctx, data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 2, rowCount)
}

func testGetSpecByID(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, conn := setupOracleSpecTest(t)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	specProtos := getTestSpecs()

	for _, proto := range specProtos {
		data := entities.OracleSpecFromProto(proto, generateTxHash(), block.VegaTime)
		assert.NoError(t, os.Upsert(ctx, data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 5, rowCount)

	got, err := os.GetSpecByID(ctx, "DEADBEEF")
	require.NoError(t, err)

	want := entities.DataSourceSpecFromProto(specProtos[0].ExternalDataSourceSpec.Spec, got.ExternalDataSourceSpec.Spec.TxHash, block.VegaTime)

	// truncate the time to microseconds as postgres doesn't support nanosecond granularity.
	want.UpdatedAt = want.UpdatedAt.Truncate(time.Microsecond)
	want.CreatedAt = want.CreatedAt.Truncate(time.Microsecond)
	s := got.ExternalDataSourceSpec.Spec
	assert.Equal(t, want, s)

	got, err = os.GetSpecByID(ctx, "beef000d")
	require.NoError(t, err)

	want = entities.DataSourceSpecFromProto(specProtos[4].ExternalDataSourceSpec.Spec, got.ExternalDataSourceSpec.Spec.TxHash, block.VegaTime)
	want.UpdatedAt = want.UpdatedAt.Truncate(time.Microsecond)
	want.CreatedAt = want.CreatedAt.Truncate(time.Microsecond)
	s = got.ExternalDataSourceSpec.Spec
	assert.Equal(t, want, s)
}

func testGetSpecByTxHash(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, conn := setupOracleSpecTest(t)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	specProtos := getTestSpecs()

	specs := make([]entities.OracleSpec, 0, len(specProtos))
	for _, proto := range specProtos {
		data := entities.OracleSpecFromProto(proto, generateTxHash(), block.VegaTime)
		assert.NoError(t, os.Upsert(ctx, data))

		specs = append(specs, *data)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 5, rowCount)

	foundSpecs, err := os.GetByTxHash(ctx, specs[0].ExternalDataSourceSpec.Spec.TxHash)
	require.NoError(t, err)

	got := foundSpecs[0]
	want := entities.DataSourceSpecFromProto(specProtos[0].ExternalDataSourceSpec.Spec, got.ExternalDataSourceSpec.Spec.TxHash, block.VegaTime)

	// truncate the time to microseconds as postgres doesn't support nanosecond granularity.
	want.UpdatedAt = want.UpdatedAt.Truncate(time.Microsecond)
	want.CreatedAt = want.CreatedAt.Truncate(time.Microsecond)
	s := got.ExternalDataSourceSpec.Spec
	assert.Equal(t, want, s)
}

func getTestSpecs() []*vegapb.OracleSpec {
	pk1 := dstypes.CreateSignerFromString("b105f00d", dstypes.SignerTypePubKey)
	pk2 := dstypes.CreateSignerFromString("0x124dd8a6044ef048614aea0aac86643a8ae1312d", dstypes.SignerTypeEthAddress)

	timeNow := uint64(time.Now().UnixNano())
	return []*vegapb.OracleSpec{
		{
			ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
				Spec: &vegapb.DataSourceSpec{
					Id:        "deadbeef",
					CreatedAt: time.Now().UnixNano(),
					UpdatedAt: time.Now().UnixNano(),
					Data: vegapb.NewDataSourceDefinition(
						vegapb.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&vega.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "Ticker",
											Type: datapb.PropertyKey_TYPE_STRING,
										},
										Conditions: []*datapb.Condition{
											{
												Operator: datapb.Condition_OPERATOR_EQUALS,
												Value:    "USDETH",
											},
										},
									},
								},
							},
						},
					),
					Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
				},
			},
		},
		{
			ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
				Spec: &vegapb.DataSourceSpec{
					Id:        "cafed00d",
					CreatedAt: time.Now().UnixNano(),
					UpdatedAt: time.Now().UnixNano(),
					Data: vegapb.NewDataSourceDefinition(
						vegapb.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&vega.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "Ticker",
											Type: datapb.PropertyKey_TYPE_STRING,
										},
										Conditions: []*datapb.Condition{
											{
												Operator: datapb.Condition_OPERATOR_EQUALS,
												Value:    "USDBTC",
											},
										},
									},
								},
							},
						},
					),
					Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
				},
			},
		},
		{
			ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
				Spec: &vegapb.DataSourceSpec{
					Id:        "deadbaad",
					CreatedAt: time.Now().UnixNano(),
					UpdatedAt: time.Now().UnixNano(),
					Data: vegapb.NewDataSourceDefinition(
						vegapb.DataSourceContentTypeOracle,
					).SetOracleConfig(
						&vega.DataSourceDefinitionExternal_Oracle{
							Oracle: &vegapb.DataSourceSpecConfiguration{
								Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "Ticker",
											Type: datapb.PropertyKey_TYPE_STRING,
										},
										Conditions: []*datapb.Condition{
											{
												Operator: datapb.Condition_OPERATOR_EQUALS,
												Value:    "USDSOL",
											},
										},
									},
								},
							},
						},
					),
					Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
				},
			},
		},
		{
			ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
				Spec: &vegapb.DataSourceSpec{
					Id:        "beefbeef",
					CreatedAt: time.Now().UnixNano(),
					UpdatedAt: time.Now().UnixNano(),
					Data: vegapb.NewDataSourceDefinition(
						vegapb.DataSourceContentTypeInternalTimeTermination,
					).SetTimeTriggerConditionConfig(
						[]*datapb.Condition{
							{
								Operator: datapb.Condition_OPERATOR_EQUALS,
								Value:    fmt.Sprintf("%v", time.Now().UnixNano()),
							},
						},
					),
					Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
				},
			},
		},
		{
			ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{
				Spec: &vegapb.DataSourceSpec{
					Id:        "beef000d",
					CreatedAt: time.Now().UnixNano(),
					UpdatedAt: time.Now().UnixNano(),
					Data: vegapb.NewDataSourceDefinition(
						vegapb.DataSourceContentTypeEthOracle,
					).SetOracleConfig(
						&vega.DataSourceDefinitionExternal_EthOracle{
							EthOracle: &vegapb.EthCallSpec{
								Address: "some-eth-address",
								Abi:     "{\"string-value\"}",
								Method:  "test-method",
								Args: []*structpb.Value{
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "string-arg",
										},
									},
								},
								Trigger: &vegapb.EthCallTrigger{
									Trigger: &vegapb.EthCallTrigger_TimeTrigger{
										TimeTrigger: &vegapb.EthTimeTrigger{
											Initial: &timeNow,
										},
									},
								},
								RequiredConfirmations: 256,
								Filters: []*datapb.Filter{
									{
										Key: &datapb.PropertyKey{
											Name: "test-key-name-0",
											Type: dstypes.SpecPropertyKeyType(2),
										},
									},
								},
							},
						},
					),
					Status: vegapb.DataSourceSpec_STATUS_ACTIVE,
				},
			},
		},
	}
}

func TestOracleSpec_GetSpecsWithCursorPagination(t *testing.T) {
	t.Run("should return the request spec of spec id is requested", testOracleSpecPaginationGetSpecID)
	t.Run("should return all specs if no spec id and no pagination is provided", testOracleSpecPaginationNoPagination)
	t.Run("should return the first page if no spec id and first is provided", testOracleSpecPaginationFirst)
	t.Run("should return the last page if no spec id and last is provided", testOracleSpecPaginationLast)
	t.Run("should return the requested page if no spec id and first and after is provided", testOracleSpecPaginationFirstAfter)
	t.Run("should return the requested page if no spec id and last and before is provided", testOracleSpecPaginationLastBefore)

	t.Run("should return all specs if no spec id and no pagination is provided - newest first", testOracleSpecPaginationNoPaginationNewestFirst)
	t.Run("should return the first page if no spec id and first is provided - newest first", testOracleSpecPaginationFirstNewestFirst)
	t.Run("should return the last page if no spec id and last is provided - newest first", testOracleSpecPaginationLastNewestFirst)
	t.Run("should return the requested page if no spec id and first and after is provided - newest first", testOracleSpecPaginationFirstAfterNewestFirst)
	t.Run("should return the requested page if no spec id and last and before is provided - newest first", testOracleSpecPaginationLastBeforeNewestFirst)
}

func createOracleSpecPaginationTestData(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, os *sqlstore.OracleSpec) []entities.OracleSpec {
	t.Helper()
	specs := make([]entities.OracleSpec, 0, 10)

	block := addTestBlockForTime(t, ctx, bs, time.Now().Truncate(time.Second))

	for i := 0; i < 10; i++ {
		pubKey := dstypes.CreateSignerFromString(helpers.GenerateID(), dstypes.SignerTypePubKey)

		spec := entities.OracleSpec{
			ExternalDataSourceSpec: &entities.ExternalDataSourceSpec{
				Spec: &entities.DataSourceSpec{
					ID:        entities.SpecID(fmt.Sprintf("deadbeef%02d", i+1)),
					CreatedAt: time.Now().Truncate(time.Microsecond),
					UpdatedAt: time.Now().Truncate(time.Microsecond),
					Data: &entities.DataSourceDefinition{
						DataSourceDefinition: &vegapb.DataSourceDefinition{
							SourceType: &vegapb.DataSourceDefinition_External{
								External: &vegapb.DataSourceDefinitionExternal{
									SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
										Oracle: &vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{
												{
													Signer: &datapb.Signer_PubKey{
														PubKey: pubKey.IntoProto().GetPubKey(),
													},
												},
											},
											Filters: nil,
										},
									},
								},
							},
						},
					},
					Status:   entities.OracleSpecActive,
					VegaTime: block.VegaTime,
				},
			},
		}

		err := os.Upsert(ctx, &spec)
		require.NoError(t, err)
		specs = append(specs, spec)
	}

	return specs
}

func testOracleSpecPaginationGetSpecID(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "deadbeef05", entities.CursorPagination{})
	require.NoError(t, err)

	assert.Equal(t, specs[4], got[0])
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     specs[4].Cursor().Encode(),
		EndCursor:       specs[4].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", entities.CursorPagination{})
	require.NoError(t, err)

	assert.Equal(t, specs, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirst(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[2].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLast(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     specs[7].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirstAfter(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	first := int32(3)
	after := specs[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[3].Cursor().Encode(),
		EndCursor:       specs[5].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLastBefore(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	last := int32(3)
	before := specs[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[4].Cursor().Encode(),
		EndCursor:       specs[6].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", entities.CursorPagination{NewestFirst: true})
	require.NoError(t, err)

	assert.Equal(t, specs, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirstNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[2].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLastNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     specs[7].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirstAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	first := int32(3)
	after := specs[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[3].Cursor().Encode(),
		EndCursor:       specs[5].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLastBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	bs, os, _ := setupOracleSpecTest(t)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	last := int32(3)
	before := specs[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[4].Cursor().Encode(),
		EndCursor:       specs[6].Cursor().Encode(),
	}, pageInfo)
}
