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
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addNetParam(t *testing.T, ns *sqlstore.NetworkParameters, key, value string, block entities.Block) entities.NetworkParameter {
	p := entities.NetworkParameter{
		Key:      key,
		Value:    value,
		VegaTime: block.VegaTime,
	}
	ns.Add(context.Background(), p)
	return p
}

func TestNetParams(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	netParamStore := sqlstore.NewNetworkParameters(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, blockStore)
	block2 := addTestBlock(t, blockStore)

	param1a := addNetParam(t, netParamStore, "foo", "bar", block1)
	param1b := addNetParam(t, netParamStore, "foo", "baz", block1)
	param2a := addNetParam(t, netParamStore, "cake", "apples", block1)
	param2b := addNetParam(t, netParamStore, "cake", "banana", block2)

	_ = param1a
	_ = param2a

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.NetworkParameter{param2b, param1b}
		pagination := entities.CursorPagination{}
		actual, _, err := netParamStore.GetAll(ctx, pagination)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
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
	defer DeleteEverything()
	ps, parameters := setupNetworkParameterPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	ps, parameters := setupNetworkParameterPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	ps, parameters := setupNetworkParameterPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	ps, parameters := setupNetworkParameterPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	after := parameters[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	ps, parameters := setupNetworkParameterPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	before := parameters[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := ps.GetAll(timeoutCtx, pagination)
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

func setupNetworkParameterPaginationTest(t *testing.T) (*sqlstore.NetworkParameters, []entities.NetworkParameter) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewNetworkParameters(connectionSource)

	blockTime := time.Date(2022, 7, 27, 8, 0, 0, 0, time.Local)
	parameters := make([]entities.NetworkParameter, 20)

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, bs, blockTime)
		id := int64(i + 1)
		parameters[i] = addNetParam(t, ps, fmt.Sprintf("key%02d", id), fmt.Sprintf("value%02d", id), block)
	}

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, bs, blockTime)
		id := int64(i + 1)
		parameters[10+i] = addNetParam(t, ps, fmt.Sprintf("key%02d", id), fmt.Sprintf("value%02d", id+10), block)
	}
	return ps, parameters
}
