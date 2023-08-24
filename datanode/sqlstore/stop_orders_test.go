package sqlstore_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/georgysavva/scany/pgxscan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/ptr"
)

type testStopOrderInputs struct {
	orderID        string
	vegaTime       time.Time
	createdAt      time.Time
	triggerPrice   string
	partyID        entities.PartyID
	marketID       entities.MarketID
	status         entities.StopOrderStatus
	expiryStrategy entities.StopOrderExpiryStrategy
}

func generateTestStopOrders(t *testing.T, testOrders []testStopOrderInputs) []entities.StopOrder {
	t.Helper()
	orders := make([]entities.StopOrder, 0)

	for i, o := range testOrders {
		so := entities.StopOrder{
			ID:               entities.StopOrderID(o.orderID),
			ExpiryStrategy:   o.expiryStrategy,
			TriggerDirection: 0,
			Status:           o.status,
			CreatedAt:        o.createdAt,
			UpdatedAt:        nil,
			OrderID:          "",
			TriggerPrice:     ptr.From(o.triggerPrice),
			PartyID:          o.partyID,
			MarketID:         o.marketID,
			VegaTime:         o.vegaTime,
			SeqNum:           uint64(i),
			TxHash:           txHashFromString(fmt.Sprintf("%s-%s-%03d", o.orderID, o.vegaTime.String(), i)),
			Submission: &commandspb.OrderSubmission{
				MarketId:    o.marketID.String(),
				Price:       "100",
				Size:        uint64(100 + i),
				Side:        entities.SideBuy,
				TimeInForce: entities.OrderTimeInForceUnspecified,
				ExpiresAt:   0,
				Type:        entities.OrderTypeMarket,
				Reference:   o.orderID,
			},
		}
		orders = append(orders, so)
	}

	return orders
}

func TestStopOrders_Add(t *testing.T) {
	so := sqlstore.NewStopOrders(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)

	ctx, rollback := tempTransaction(t)
	defer rollback()

	blocks := []entities.Block{
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
	}

	parties := []entities.Party{
		addTestParty(t, ctx, ps, blocks[0]),
		addTestParty(t, ctx, ps, blocks[0]),
		addTestParty(t, ctx, ps, blocks[0]),
	}

	markets := helpers.GenerateMarkets(t, ctx, 3, blocks[0], ms)

	inputs := make([]testStopOrderInputs, 0)

	for _, b := range blocks {
		for _, p := range parties {
			for _, m := range markets {
				inputs = append(inputs, testStopOrderInputs{
					orderID:        helpers.GenerateID(),
					vegaTime:       b.VegaTime,
					createdAt:      b.VegaTime,
					triggerPrice:   "100",
					partyID:        p.ID,
					marketID:       m.ID,
					status:         entities.StopOrderStatusUnspecified,
					expiryStrategy: entities.StopOrderExpiryStrategyUnspecified,
				})
			}
		}
	}

	stopOrders := generateTestStopOrders(t, inputs)

	t.Run("add should batch orders and not insert the records", func(t *testing.T) {
		for _, o := range stopOrders {
			err := so.Add(o)
			require.NoError(t, err)
		}

		rows, err := connectionSource.Connection.Query(ctx, "select * from stop_orders")
		require.NoError(t, err)
		assert.False(t, rows.Next())

		t.Run("and insert them when flush is called", func(t *testing.T) {
			orders, err := so.Flush(ctx)
			require.NoError(t, err)
			assert.Len(t, orders, len(stopOrders))

			var results []entities.StopOrder
			err = pgxscan.Select(ctx, connectionSource.Connection, &results, "select * from stop_orders")
			require.NoError(t, err)
			assert.Len(t, results, len(stopOrders))
			assert.ElementsMatch(t, results, orders)
		})
	})
}

func TestStopOrders_Get(t *testing.T) {
	so := sqlstore.NewStopOrders(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)

	ctx, rollback := tempTransaction(t)
	defer rollback()

	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)

	party := addTestParty(t, ctx, ps, block)

	markets := helpers.GenerateMarkets(t, ctx, 1, block, ms)

	orderID := helpers.GenerateID()
	stopOrders := generateTestStopOrders(t, []testStopOrderInputs{
		{
			orderID:        orderID,
			vegaTime:       block.VegaTime,
			createdAt:      block.VegaTime,
			triggerPrice:   "100",
			partyID:        party.ID,
			marketID:       markets[0].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyUnspecified,
		},
		{
			orderID:        orderID,
			vegaTime:       block2.VegaTime,
			createdAt:      block.VegaTime,
			triggerPrice:   "100",
			partyID:        party.ID,
			marketID:       markets[0].ID,
			status:         entities.StopOrderStatusTriggered,
			expiryStrategy: entities.StopOrderExpiryStrategyUnspecified,
		},
	})

	for i := range stopOrders {
		err := so.Add(stopOrders[i])
		require.NoError(t, err)
	}

	want, err := so.Flush(ctx)
	require.NoError(t, err)

	t.Run("Get should return an error if the order ID does not exist", func(t *testing.T) {
		got, err := so.GetStopOrder(ctx, "deadbeef")
		require.Error(t, err)
		assert.Equal(t, entities.StopOrder{}, got)
	})

	t.Run("Get should return the order if the order ID does exist", func(t *testing.T) {
		got, err := so.GetStopOrder(ctx, orderID)
		require.NoError(t, err)
		assert.Equal(t, want[1], got)
	})
}

func TestStopOrders_ListStopOrders(t *testing.T) {
	so := sqlstore.NewStopOrders(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)

	// ctx, rollback := tempTransaction(t)
	// defer rollback()

	ctx := context.Background()

	blocks := []entities.Block{
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
		addTestBlock(t, ctx, bs),
	}

	parties := []entities.Party{
		addTestParty(t, ctx, ps, blocks[0]),
		addTestParty(t, ctx, ps, blocks[0]),
		addTestParty(t, ctx, ps, blocks[0]),
		addTestParty(t, ctx, ps, blocks[0]),
	}

	markets := helpers.GenerateMarkets(t, ctx, 10, blocks[0], ms)
	orderIDs := []string{
		"deadbeef01",
		"deadbeef02",
		"deadbeef03",
		"deadbeef04",
		"deadbeef05",
		"deadbeef06",
		"deadbeef07",
		"deadbeef08",
		"deadbeef09",
		"deadbeef10",
		"deadbeef11",
		"deadbeef12",
	}

	stopOrders := generateTestStopOrders(t, []testStopOrderInputs{
		{ // 0
			orderID:        orderIDs[0],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[0].ID,
			marketID:       markets[0].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategyUnspecified,
		},
		{ // 1
			orderID:        orderIDs[1],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[1].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 2
			orderID:        orderIDs[2],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[2].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 3
			orderID:        orderIDs[3],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[3].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 4
			orderID:        orderIDs[4],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[4].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 5
			orderID:        orderIDs[5],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[5].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 6
			orderID:        orderIDs[6],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[6].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 7
			orderID:        orderIDs[7],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[7].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 8
			orderID:        orderIDs[8],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[8].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 9
			orderID:        orderIDs[9],
			vegaTime:       blocks[0].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[9].ID,
			status:         entities.StopOrderStatusUnspecified,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		// block 2
		{ // 10
			orderID:        orderIDs[0],
			vegaTime:       blocks[1].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[0].ID,
			marketID:       markets[0].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyUnspecified,
		},
		{ // 11
			orderID:        orderIDs[2],
			vegaTime:       blocks[1].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[2].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 12
			orderID:        orderIDs[3],
			vegaTime:       blocks[1].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[3].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 13
			orderID:        orderIDs[4],
			vegaTime:       blocks[1].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[4].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 14
			orderID:        orderIDs[6],
			vegaTime:       blocks[1].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[6].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 15
			orderID:        orderIDs[7],
			vegaTime:       blocks[1].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[7].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		// block 3
		{ // 16
			orderID:        orderIDs[0],
			vegaTime:       blocks[2].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[0].ID,
			marketID:       markets[0].ID,
			status:         entities.StopOrderStatusCancelled,
			expiryStrategy: entities.StopOrderExpiryStrategyUnspecified,
		},
		{ // 17
			orderID:        orderIDs[1],
			vegaTime:       blocks[2].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[1].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 18
			orderID:        orderIDs[5],
			vegaTime:       blocks[2].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[5].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 19
			orderID:        orderIDs[8],
			vegaTime:       blocks[2].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[8].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 20
			orderID:        orderIDs[9],
			vegaTime:       blocks[2].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[9].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		// block 4
		{ // 21
			orderID:        orderIDs[1],
			vegaTime:       blocks[3].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[1].ID,
			status:         entities.StopOrderStatusExpired,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 22
			orderID:        orderIDs[2],
			vegaTime:       blocks[3].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[2].ID,
			status:         entities.StopOrderStatusExpired,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 23
			orderID:        orderIDs[3],
			vegaTime:       blocks[3].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[3].ID,
			status:         entities.StopOrderStatusTriggered,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 24
			orderID:        orderIDs[6],
			vegaTime:       blocks[3].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[6].ID,
			status:         entities.StopOrderStatusCancelled,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 25
			orderID:        orderIDs[7],
			vegaTime:       blocks[3].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[7].ID,
			status:         entities.StopOrderStatusTriggered,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		// block 5
		{ // 26
			orderID:        orderIDs[4],
			vegaTime:       blocks[4].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[4].ID,
			status:         entities.StopOrderStatusTriggered,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 27
			orderID:        orderIDs[5],
			vegaTime:       blocks[4].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[5].ID,
			status:         entities.StopOrderStatusCancelled,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 28
			orderID:        orderIDs[8],
			vegaTime:       blocks[4].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[2].ID,
			marketID:       markets[8].ID,
			status:         entities.StopOrderStatusStopped,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 29
			orderID:        orderIDs[9],
			vegaTime:       blocks[4].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[1].ID,
			marketID:       markets[9].ID,
			status:         entities.StopOrderStatusStopped,
			expiryStrategy: entities.StopOrderExpiryStrategySubmit,
		},
		{ // 30
			orderID:        orderIDs[10],
			vegaTime:       blocks[4].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[3].ID,
			marketID:       markets[8].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
		{ // 31
			orderID:        orderIDs[11],
			vegaTime:       blocks[5].VegaTime,
			createdAt:      blocks[0].VegaTime,
			triggerPrice:   "100",
			partyID:        parties[3].ID,
			marketID:       markets[9].ID,
			status:         entities.StopOrderStatusPending,
			expiryStrategy: entities.StopOrderExpiryStrategyCancels,
		},
	})

	for _, o := range stopOrders {
		require.NoError(t, so.Add(o))
	}

	saved, err := so.Flush(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, stopOrders, saved)

	t.Run("should return an error if oldest first order is requested in pagination", func(t *testing.T) {
		filter := entities.StopOrderFilter{}
		p := entities.CursorPagination{}

		_, _, err := so.ListStopOrders(ctx, filter, p)
		require.Error(t, err)
		assert.EqualError(t, err, "oldest first order query is not currently supported")
	})

	t.Run("should list the latest version of each stop order", func(t *testing.T) {
		want := []entities.StopOrder{
			stopOrders[29],
			stopOrders[28],
			stopOrders[25],
			stopOrders[24],
			stopOrders[27],
			stopOrders[26],
			stopOrders[23],
			stopOrders[22],
			stopOrders[21],
			stopOrders[16],
			stopOrders[30],
			stopOrders[31],
		}

		sort.Slice(want, func(i, j int) bool {
			return want[i].ID < want[j].ID
		})

		filter := entities.StopOrderFilter{}
		p := entities.CursorPagination{
			NewestFirst: true,
		}

		got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should paginate the data correctly", func(t *testing.T) {
		current := []entities.StopOrder{
			stopOrders[31],
			stopOrders[30],
			stopOrders[29],
			stopOrders[28],
			stopOrders[25],
			stopOrders[24],
			stopOrders[27],
			stopOrders[26],
			stopOrders[23],
			stopOrders[22],
			stopOrders[21],
			stopOrders[16],
		}

		sort.Slice(current, func(i, j int) bool {
			return current[i].ID < current[j].ID
		})

		filter := entities.StopOrderFilter{}
		first := int32(3)
		currentIndex := -1

		for {
			var (
				p                    entities.CursorPagination
				err                  error
				hasNext, hasPrevious bool
				want                 []entities.StopOrder
			)

			if currentIndex > -1 {
				p, err = entities.NewCursorPagination(&first, ptr.From(current[currentIndex].Cursor().Encode()), nil, nil, true)
				require.NoError(t, err)
				hasNext = (currentIndex + int(first)) < len(current)-1
				hasPrevious = true
			} else {
				p, err = entities.NewCursorPagination(&first, nil, nil, nil, true)
				require.NoError(t, err)
				hasNext = true
				hasPrevious = false
			}

			wantStart := currentIndex + 1
			wantEnd := wantStart + int(first)

			if wantEnd > len(current) {
				wantEnd = len(current)
			}

			want = current[wantStart:wantEnd]

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(t, err)
			assert.Equal(t, want, got)
			assert.Equal(t, entities.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: hasPrevious,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)

			currentIndex += len(got)
			if currentIndex >= len(current)-1 {
				break
			}
		}
	})

	t.Run("should filter orders", func(tt *testing.T) {
		tt.Run("by party", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				PartyIDs: []string{
					parties[0].ID.String(),
					parties[1].ID.String(),
				},
			}

			want := []entities.StopOrder{
				stopOrders[29],
				stopOrders[25],
				stopOrders[27],
				stopOrders[23],
				stopOrders[21],
				stopOrders[16],
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].PartyID == want[j].PartyID {
					return want[i].ID < want[j].ID
				}
				return want[i].PartyID < want[j].PartyID
			})

			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("by market", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				MarketIDs: []string{
					markets[0].ID.String(),
					markets[1].ID.String(),
					markets[3].ID.String(),
					markets[6].ID.String(),
					markets[7].ID.String(),
					markets[8].ID.String(),
				},
			}

			want := []entities.StopOrder{
				stopOrders[30],
				stopOrders[28],
				stopOrders[25],
				stopOrders[24],
				stopOrders[23],
				stopOrders[21],
				stopOrders[16],
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].MarketID == want[j].MarketID {
					return want[i].ID < want[j].ID
				}
				return want[i].MarketID < want[j].MarketID
			})

			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("by party and market", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				PartyIDs: []string{
					parties[1].ID.String(),
					parties[2].ID.String(),
				},
				MarketIDs: []string{
					markets[1].ID.String(),
					markets[3].ID.String(),
					markets[6].ID.String(),
					markets[7].ID.String(),
					markets[8].ID.String(),
				},
			}

			want := []entities.StopOrder{
				stopOrders[28],
				stopOrders[25],
				stopOrders[24],
				stopOrders[23],
				stopOrders[21],
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].PartyID == want[j].PartyID {
					return want[i].ID < want[j].ID
				}
				return want[i].PartyID < want[j].PartyID
			})

			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("by status", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				Statuses: []entities.StopOrderStatus{
					entities.StopOrderStatusCancelled,
					entities.StopOrderStatusTriggered,
					entities.StopOrderStatusExpired,
				},
			}

			want := []entities.StopOrder{
				stopOrders[27],
				stopOrders[26],
				stopOrders[25],
				stopOrders[24],
				stopOrders[23],
				stopOrders[22],
				stopOrders[21],
				stopOrders[16],
			}
			sort.Slice(want, func(i, j int) bool {
				return want[i].ID < want[j].ID
			})

			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("by trigger strategy", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				ExpiryStrategy: []entities.StopOrderExpiryStrategy{
					entities.StopOrderExpiryStrategyUnspecified,
					entities.StopOrderExpiryStrategyCancels,
				},
			}

			want := []entities.StopOrder{
				stopOrders[31],
				stopOrders[30],
				stopOrders[28],
				stopOrders[26],
				stopOrders[24],
				stopOrders[22],
				stopOrders[16],
			}

			sort.Slice(want, func(i, j int) bool {
				return want[i].ID < want[j].ID
			})

			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("by party and status", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				PartyIDs: []string{
					parties[1].ID.String(),
				},
				Statuses: []entities.StopOrderStatus{
					entities.StopOrderStatusTriggered,
					entities.StopOrderStatusExpired,
				},
			}

			want := []entities.StopOrder{
				stopOrders[25],
				stopOrders[23],
				stopOrders[21],
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].PartyID == want[j].PartyID {
					return want[i].ID < want[j].ID
				}
				return want[i].PartyID < want[j].PartyID
			})
			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("by market and trigger strategy", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				MarketIDs: []string{
					markets[0].ID.String(),
					markets[1].ID.String(),
					markets[3].ID.String(),
					markets[6].ID.String(),
					markets[7].ID.String(),
					markets[8].ID.String(),
				},
				ExpiryStrategy: []entities.StopOrderExpiryStrategy{
					entities.StopOrderExpiryStrategyCancels,
				},
			}

			want := []entities.StopOrder{
				stopOrders[30],
				stopOrders[28],
				stopOrders[24],
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].MarketID == want[j].MarketID {
					return want[i].ID < want[j].ID
				}
				return want[i].MarketID < want[j].MarketID
			})

			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)

			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)
			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[len(want)-1].Cursor().Encode(),
			}, pageInfo)
		})

		tt.Run("live only", func(ts *testing.T) {
			filter := entities.StopOrderFilter{
				LiveOnly: true,
			}

			want := []entities.StopOrder{
				stopOrders[31],
				stopOrders[30],
			}
			sort.Slice(want, func(i, j int) bool {
				return want[i].ID < want[j].ID
			})
			p, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
			require.NoError(ts, err)
			got, pageInfo, err := so.ListStopOrders(ctx, filter, p)
			require.NoError(ts, err)

			assert.Equal(ts, want, got)
			assert.Equal(ts, entities.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     want[0].Cursor().Encode(),
				EndCursor:       want[1].Cursor().Encode(),
			}, pageInfo)
		})
	})
}
