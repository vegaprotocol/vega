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
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/jackc/pgx/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStakeLinkingStore(t *testing.T) {
	t.Run("Upsert should add a stake linking record if it doesn't exist in the current block", testUpsertShouldAddNewInBlock)
	t.Run("Upsert should update a stake linking record if it already exists in the current block", testUpsertShouldUpdateExistingInBlock)
	t.Run("GetStake should return the most current version of each stake linking record and calculate the total stake available", testGetStake)
}

func setupStakeLinkingTest(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.StakeLinking, *pgx.Conn) {
	t.Helper()
	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	sl := sqlstore.NewStakeLinking(connectionSource)

	config := NewTestConfig(testDBPort)
	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return bs, sl, conn
}

func testUpsertShouldAddNewInBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, sl, conn := setupStakeLinkingTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from stake_linking").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	stakingProtos := getStakingProtos()

	proto := stakingProtos[0]
	data, err := entities.StakeLinkingFromProto(proto, block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, sl.Upsert(context.Background(), data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from stake_linking").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testUpsertShouldUpdateExistingInBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, sl, conn := setupStakeLinkingTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from stake_linking").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	stakingProtos := getStakingProtos()

	for _, proto := range stakingProtos {
		data, err := entities.StakeLinkingFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, sl.Upsert(context.Background(), data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from stake_linking").Scan(&rowCount))
	assert.Equal(t, 2, rowCount)
}

func testGetStake(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, sl, conn := setupStakeLinkingTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from stake_linking").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	stakingProtos := getStakingProtos()

	for _, proto := range stakingProtos {
		data, err := entities.StakeLinkingFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, sl.Upsert(context.Background(), data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from stake_linking").Scan(&rowCount))
	assert.Equal(t, 2, rowCount)

	partyID := entities.NewPartyID("cafed00d")

	currentBalance, links, _, err := sl.GetStake(ctx, partyID, entities.OffsetPagination{})
	require.NoError(t, err)
	want := num.NewUint(30002)
	assert.True(t, want.EQ(currentBalance))
	assert.Equal(t, 2, len(links))
}

func getStakingProtos() []*eventspb.StakeLinking {
	return []*eventspb.StakeLinking{
		{
			Id:              "deadbeef",
			Type:            eventspb.StakeLinking_TYPE_LINK,
			Ts:              time.Now().Unix(),
			Party:           "cafed00d",
			Amount:          "10000",
			Status:          eventspb.StakeLinking_STATUS_ACCEPTED,
			FinalizedAt:     time.Now().UnixNano(),
			TxHash:          "0xfe179560b9d0cc44c5fea54c2167c1cee7ccfcabf294752a4f43fb64ddffda85",
			BlockHeight:     1000000,
			BlockTime:       0,
			LogIndex:        100000,
			EthereumAddress: "TEST",
		},
		{
			Id:              "deadbeef",
			Type:            eventspb.StakeLinking_TYPE_LINK,
			Ts:              time.Now().Unix(),
			Party:           "cafed00d",
			Amount:          "10001",
			Status:          eventspb.StakeLinking_STATUS_ACCEPTED,
			FinalizedAt:     time.Now().UnixNano(),
			TxHash:          "0xfe179560b9d0cc44c5fea54c2167c1cee7ccfcabf294752a4f43fb64ddffda85",
			BlockHeight:     1000000,
			BlockTime:       0,
			LogIndex:        100000,
			EthereumAddress: "TEST",
		},
		{
			Id:              "deadbaad",
			Type:            eventspb.StakeLinking_TYPE_LINK,
			Ts:              time.Now().Unix(),
			Party:           "cafed00d",
			Amount:          "20001",
			Status:          eventspb.StakeLinking_STATUS_ACCEPTED,
			FinalizedAt:     time.Now().UnixNano(),
			TxHash:          "0xfe179560b9d0cc44c5fea54c2167c1cee7ccfcabf294752a4f43fb64ddffda85",
			BlockHeight:     1000000,
			BlockTime:       0,
			LogIndex:        100000,
			EthereumAddress: "TEST",
		},
	}
}

func TestStakeLinkingPagination(t *testing.T) {
	t.Run("should return all stake linkings if no pagination is specified", testStakeLinkingPaginationNoPagination)
	t.Run("should return first page of stake linkings if first is provided", testStakeLinkingPaginationFirst)
	t.Run("should return last page of stake linkings if last is provided", testStakeLinkingPaginationLast)
	t.Run("should return specified page of stake linkings if first and after is specified", testStakeLinkingPaginationFirstAndAfter)
	t.Run("should return specified page of stake linkings if last and before is specified", testStakeLinkingPaginationLastAndBefore)

	t.Run("should return all stake linkings if no pagination is specified - newest first", testStakeLinkingPaginationNoPaginationNewestFirst)
	t.Run("should return first page of stake linkings if first is provided - newest first", testStakeLinkingPaginationFirstNewestFirst)
	t.Run("should return last page of stake linkings if last is provided - newest first", testStakeLinkingPaginationLastNewestFirst)
	t.Run("should return specified page of stake linkings if first and after is specified - newest first", testStakeLinkingPaginationFirstAndAfterNewestFirst)
	t.Run("should return specified page of stake linkings if last and before is specified - newest first", testStakeLinkingPaginationLastAndBeforeNewestFirst)
}

func testStakeLinkingPaginationNoPagination(t *testing.T) {
	DeleteEverything()
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := links[10:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationFirst(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := links[10:13]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationLast(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := links[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationFirstAndAfter(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	after := links[12].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := links[13:16]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationLastAndBefore(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	before := links[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := links[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationNoPaginationNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(links[10:])
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationFirstNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(links[10:])[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationLastNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(links[10:])[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationFirstAndAfterNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	after := links[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(links[10:])[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testStakeLinkingPaginationLastAndBeforeNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ls, links := setupStakeLinkingPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	before := links[12].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("cafed00d")

	_, got, pageInfo, err := ls.GetStake(timeoutCtx, partyID, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(links[10:])[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func addStakeLinking(t *testing.T, ctx context.Context, ls *sqlstore.StakeLinking, id string, partyID string, logIndex int64, block entities.Block) entities.StakeLinking {
	t.Helper()
	l := entities.StakeLinking{
		ID:                 entities.NewStakeLinkingID(id),
		StakeLinkingType:   entities.StakeLinkingTypeLink,
		EthereumTimestamp:  block.VegaTime,
		PartyID:            entities.NewPartyID(partyID),
		Amount:             decimal.NewFromFloat(1),
		StakeLinkingStatus: entities.StakeLinkingStatusAccepted,
		FinalizedAt:        block.VegaTime,
		TxHash:             generateID(),
		LogIndex:           logIndex,
		EthereumAddress:    "0xfe179560b9d0cc44c5fea54c2167c1cee7ccfcabf294752a4f43fb64ddffda85",
		VegaTime:           block.VegaTime,
	}

	ls.Upsert(ctx, &l)

	return l
}

func setupStakeLinkingPaginationTest(t *testing.T) (*sqlstore.StakeLinking, []entities.StakeLinking) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ls := sqlstore.NewStakeLinking(connectionSource)

	blockTime := time.Date(2022, 7, 27, 8, 0, 0, 0, time.Local)
	linkings := make([]entities.StakeLinking, 20)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	partyID := "cafed00d"
	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, bs, blockTime)
		id := int64(i + 1)
		linkingID := fmt.Sprintf("deadbeef%02d", id)

		linkings[i] = addStakeLinking(t, ctx, ls, linkingID, partyID, id, block)
	}

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, bs, blockTime)
		id := int64(i + 1)
		linkingID := fmt.Sprintf("deadbeef%02d", id)
		linkings[10+i] = addStakeLinking(t, ctx, ls, linkingID, partyID, id+10, block)
	}
	return ls, linkings
}
