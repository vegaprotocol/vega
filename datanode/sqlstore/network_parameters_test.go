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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addNetParam(t *testing.T, ctx context.Context, ns *sqlstore.NetworkParameters, key, value string, block entities.Block) entities.NetworkParameter {
	t.Helper()
	p := entities.NetworkParameter{
		Key:      key,
		Value:    value,
		VegaTime: block.VegaTime,
	}
	ns.Add(ctx, p)
	return p
}

func TestNetParams(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	netParamStore := sqlstore.NewNetworkParameters(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)
	block2 := addTestBlock(t, ctx, blockStore)

	param1a := addNetParam(t, ctx, netParamStore, "foo", "bar", block1)
	param1b := addNetParam(t, ctx, netParamStore, "foo", "baz", block1)
	param2a := addNetParam(t, ctx, netParamStore, "cake", "apples", block1)
	param2b := addNetParam(t, ctx, netParamStore, "cake", "banana", block2)

	_ = param1a
	_ = param2a

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.NetworkParameter{param2b, param1b}
		pagination := entities.CursorPagination{}
		actual, _, err := netParamStore.GetAll(ctx, pagination)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("GetByKey", func(t *testing.T) {
		param, err := netParamStore.GetByKey(ctx, "foo")
		require.NoError(t, err)
		assert.Equal(t, "baz", param.Value)

		_, err = netParamStore.GetByKey(ctx, "foo1")
		assert.Equal(t, entities.ErrNotFound, err)
	})
}

func TestNetworkParameterPagination(t *testing.T) {
	t.Run("should return all network parameters if no pagination is specified", testNetworkParameterPaginationNoPagination)
	t.Run("should return first page of network parameters if first is provided", testNetworkParameterPaginationFirst)
	t.Run("should return last page of network parameters if last is provided", testNetworkParameterPaginationLast)
	t.Run("should return specified page of network parameters if first and after is specified", testNetworkParameterPaginationFirstAndAfter)
	t.Run("should return specified page of network parameters if last and before is specified", testNetworkParameterPaginationLastAndBefore)
}

func testNetworkParameterPaginationNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps, parameters := setupNetworkParameterPaginationTest(t, ctx)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := parameters[10:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testNetworkParameterPaginationFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps, parameters := setupNetworkParameterPaginationTest(t, ctx)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := parameters[10:13]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNetworkParameterPaginationLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps, parameters := setupNetworkParameterPaginationTest(t, ctx)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := parameters[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNetworkParameterPaginationFirstAndAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps, parameters := setupNetworkParameterPaginationTest(t, ctx)

	first := int32(3)
	after := parameters[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := parameters[13:16]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNetworkParameterPaginationLastAndBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	ps, parameters := setupNetworkParameterPaginationTest(t, ctx)

	last := int32(3)
	before := parameters[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := parameters[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func setupNetworkParameterPaginationTest(t *testing.T, ctx context.Context) (*sqlstore.NetworkParameters, []entities.NetworkParameter) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewNetworkParameters(connectionSource)

	blockTime := time.Date(2022, 7, 27, 8, 0, 0, 0, time.Local)
	parameters := make([]entities.NetworkParameter, 20)

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, ctx, bs, blockTime)
		id := int64(i + 1)
		parameters[i] = addNetParam(t, ctx, ps, fmt.Sprintf("key%02d", id), fmt.Sprintf("value%02d", id), block)
	}

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, ctx, bs, blockTime)
		id := int64(i + 1)
		parameters[10+i] = addNetParam(t, ctx, ps, fmt.Sprintf("key%02d", id), fmt.Sprintf("value%02d", id+10), block)
	}
	return ps, parameters
}
