package liquidity_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/liquidity/v2/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	mmocks "code.vegaprotocol.io/vega/core/execution/common/mocks"
)

const partyID = "lp-party-1"

type testEngine struct {
	ctrl         *gomock.Controller
	marketID     string
	tsvc         *stubs.TimeStub
	broker       *bmocks.MockBroker
	riskModel    *mocks.MockRiskModel
	priceMonitor *mocks.MockPriceMonitor
	orderbook    *mocks.MockOrderBook
	auctionState *mmocks.MockAuctionState
	engine       *liquidity.Engine
	stateVar     *stubs.StateVarStub
}

func newTestEngine(t *testing.T) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)

	log := logging.NewTestLogger()
	tsvc := stubs.NewTimeStub()

	broker := bmocks.NewMockBroker(ctrl)
	risk := mocks.NewMockRiskModel(ctrl)
	monitor := mocks.NewMockPriceMonitor(ctrl)
	orderbook := mocks.NewMockOrderBook(ctrl)
	market := "market-id"
	asset := "asset-id"
	liquidityConfig := liquidity.NewDefaultConfig()
	stateVarEngine := stubs.NewStateVar()
	risk.EXPECT().GetProjectionHorizon().AnyTimes()

	auctionState := mmocks.NewMockAuctionState(ctrl)

	targetStakeFunc := func() *num.Uint {
		return num.UintOne()
	}

	engine := liquidity.NewEngine(liquidityConfig,
		log, tsvc, broker, risk, monitor, orderbook, auctionState, asset, market, stateVarEngine,
		num.NewDecimalFromFloat(1), // positionFactor
		num.DecimalFromInt64(1),    // stakeToCcyVolume
		num.DecimalFromFloat(0.2),  // priceRange
		num.DecimalFromFloat(0.5),  // commitmentMinTimeFraction
		num.DecimalFromFloat(1),    // slaCompetitionFactor
		num.DecimalFromFloat(2),    // nonPerformanceBondPenaltySlope
		num.DecimalFromFloat(0.5),  // nonPerformanceBondPenaltyMax
		4,                          // performanceHysteresisEpochs
		targetStakeFunc,
	)

	return &testEngine{
		ctrl:         ctrl,
		marketID:     market,
		tsvc:         tsvc,
		broker:       broker,
		riskModel:    risk,
		priceMonitor: monitor,
		orderbook:    orderbook,
		auctionState: auctionState,
		engine:       engine,
		stateVar:     stateVarEngine,
	}
}

type stubIDGen struct {
	calls int
}

func (s *stubIDGen) NextID() string {
	s.calls++
	return hex.EncodeToString([]byte(fmt.Sprintf("deadb33f%d", s.calls)))
}

func toPoint[T any](v T) *T {
	return &v
}

func generateOrders(idGen stubIDGen, marketID string, buys, sells []uint64) []*types.Order {
	newOrder := func(price uint64, side types.Side) *types.Order {
		return &types.Order{
			ID:        idGen.NextID(),
			MarketID:  marketID,
			Party:     partyID,
			Side:      side,
			Price:     num.NewUint(price),
			Remaining: price,
			Status:    types.OrderStatusActive,
		}
	}

	orders := []*types.Order{}
	for _, price := range buys {
		orders = append(orders, newOrder(price, types.SideBuy))
	}

	for _, price := range sells {
		orders = append(orders, newOrder(price, types.SideSell))
	}

	return orders
}

func TestSLAPerformanceSingleEpochFeePenalty(t *testing.T) {
	testCases := []struct {
		desc string

		// represents list of active orders by a party on a book in a given block
		buyOrdersPerBlock   [][]uint64
		sellsOrdersPerBlock [][]uint64

		epochLength int

		// optional net params to set
		slaCompetitionFactor        *num.Decimal
		commitmentMinTimeFraction   *num.Decimal
		priceRange                  *num.Decimal
		performanceHysteresisEpochs *uint

		// expected result
		expectedPenalty num.Decimal
	}{
		{
			desc:                 "Meets commitment with fraction_of_time_on_book=0.75 and slaCompetitionFactor=1, 0042-LIQF-037",
			epochLength:          4,
			buyOrdersPerBlock:    [][]uint64{{15, 15, 17, 18}, {15, 15, 17, 18}, {15, 15, 17, 18}, {}},
			sellsOrdersPerBlock:  [][]uint64{{12, 12, 12}, {12, 12, 12}, {12, 12, 12}, {}},
			slaCompetitionFactor: toPoint(num.DecimalFromFloat(1)),
			expectedPenalty:      num.DecimalFromFloat(0.5),
		},
		{
			desc:                 "Meets commitment with fraction_of_time_on_book=0.75 and slaCompetitionFactor=1, 0042-LIQF-038",
			epochLength:          4,
			buyOrdersPerBlock:    [][]uint64{{15, 15, 17, 18}, {}, {15, 15, 17, 18}, {15, 15, 17, 18}},
			sellsOrdersPerBlock:  [][]uint64{{12, 12, 12}, {}, {12, 12, 12}, {12, 12, 12}},
			slaCompetitionFactor: toPoint(num.DecimalFromFloat(1)),
			expectedPenalty:      num.DecimalFromFloat(0.5),
		},
		{
			desc:                 "Meets commitment with fraction_of_time_on_book=0.75 and slaCompetitionFactor=0, 0042-LIQF-041",
			epochLength:          4,
			buyOrdersPerBlock:    [][]uint64{{15, 15, 17, 18}, {15, 15, 17, 18}, {15, 15, 17, 18}, {}},
			sellsOrdersPerBlock:  [][]uint64{{12, 12, 12}, {12, 12, 12}, {12, 12, 12}, {}},
			slaCompetitionFactor: toPoint(num.DecimalFromFloat(0)),
			expectedPenalty:      num.DecimalFromFloat(0.0),
		},
		{
			desc:                 "Meets commitment with fraction_of_time_on_book=0.75 and slaCompetitionFactor=0.5, 0042-LIQF-042",
			epochLength:          4,
			buyOrdersPerBlock:    [][]uint64{{15, 15, 17, 18}, {15, 15, 17, 18}, {15, 15, 17, 18}, {}},
			sellsOrdersPerBlock:  [][]uint64{{12, 12, 12}, {12, 12, 12}, {12, 12, 12}, {}},
			slaCompetitionFactor: toPoint(num.DecimalFromFloat(0.5)),
			expectedPenalty:      num.DecimalFromFloat(0.25),
		},
		{
			desc:                        "Meets commitment with fraction_of_time_on_book=1 and performanceHysteresisEpochs=0, 0042-LIQF-035",
			performanceHysteresisEpochs: toPoint[uint](0),
			epochLength:                 3,
			buyOrdersPerBlock:           [][]uint64{{15, 15, 17, 18}, {15, 15, 17, 18}, {15, 15, 17, 18}},
			sellsOrdersPerBlock:         [][]uint64{{12, 12, 12}, {12, 12, 12}, {12, 12, 12}},
			expectedPenalty:             num.DecimalFromFloat(0),
		},
		{
			desc:                        "Does not meet commitment with fraction_of_time_on_book=0.5 and performanceHysteresisEpochs=0, 0042-LIQF-036",
			performanceHysteresisEpochs: toPoint[uint](0),
			epochLength:                 6,
			buyOrdersPerBlock:           [][]uint64{{15, 15, 17, 18}, {}, {15, 15, 17, 18}, {}, {}, {15, 15, 17, 18}},
			sellsOrdersPerBlock:         [][]uint64{{12, 12, 12}, {}, {12, 12, 12}, {}, {}, {12, 12, 12}},
			expectedPenalty:             num.DecimalFromFloat(1),
		},
	}

	for i := 0; i < 2; i++ {
		inAuction := i != 0

		for _, tC := range testCases {
			desc := tC.desc
			if inAuction {
				desc = fmt.Sprintf("%s in auction", tC.desc)
			}
			t.Run(desc, func(t *testing.T) {
				te := newTestEngine(t)

				// set the net params
				if tC.slaCompetitionFactor != nil {
					te.engine.OnSlaCompetitionFactorUpdate(*tC.slaCompetitionFactor)
				}
				if tC.commitmentMinTimeFraction != nil {
					te.engine.OnCommitmentMinTimeFractionUpdate(*tC.commitmentMinTimeFraction)
				}
				if tC.priceRange != nil {
					te.engine.OnPriceRangeUpdate(*tC.priceRange)
				}
				if tC.performanceHysteresisEpochs != nil {
					te.engine.OnPerformanceHysteresisEpochsUpdate(*tC.performanceHysteresisEpochs)
				}

				idGen := &stubIDGen{}
				ctx := context.Background()
				party := "lp-party-1"

				te.broker.EXPECT().Send(gomock.Any()).AnyTimes()

				lps := &types.LiquidityProvisionSubmission{
					MarketID:         te.marketID,
					CommitmentAmount: num.NewUint(100),
					Fee:              num.NewDecimalFromFloat(0.5),
					Reference:        fmt.Sprintf("provision-by-%s", party),
				}

				err := te.engine.SubmitLiquidityProvision(ctx, lps, party, idGen)
				require.NoError(t, err)

				te.auctionState.EXPECT().InAuction().Return(inAuction).AnyTimes()

				te.orderbook.EXPECT().GetLastTradedPrice().Return(num.NewUint(15)).AnyTimes()
				te.orderbook.EXPECT().GetIndicativePrice().Return(num.NewUint(15)).AnyTimes()

				te.orderbook.EXPECT().GetBestStaticBidPrice().Return(num.NewUint(20), nil).AnyTimes()
				te.orderbook.EXPECT().GetBestStaticAskPrice().Return(num.NewUint(10), nil).AnyTimes()
				orders := []*types.Order{}
				te.orderbook.EXPECT().GetOrdersPerParty(party).DoAndReturn(func(party string) []*types.Order {
					return orders
				}).AnyTimes()

				epochLength := time.Duration(tC.epochLength) * time.Second
				epochStart := time.Now().Add(-epochLength)
				epochEnd := epochStart.Add(epochLength)

				orders = generateOrders(*idGen, te.marketID, tC.buyOrdersPerBlock[0], tC.sellsOrdersPerBlock[0])

				te.engine.ResetSLAEpoch(epochStart)
				txs := []liquidity.TX{{ID: "1"}, {ID: "2"}, {ID: "3"}}

				k := te.engine.GenerateKSla(txs)

				for i := 0; i < tC.epochLength; i++ {
					orders = generateOrders(*idGen, te.marketID, tC.buyOrdersPerBlock[i], tC.sellsOrdersPerBlock[i])

					te.tsvc.SetTime(epochStart.Add(time.Duration(i) * time.Second))
					te.engine.BeginBlock(txs)
					te.engine.TxProcessed(k)
				}

				te.engine.CalculateSLAPenalties(epochEnd)

				sla := te.engine.GetSLAPenalties()[party]

				fmt.Printf("actual penalty: %s, expected penalty: %s \n", sla.Fee, tC.expectedPenalty)
				require.True(t, sla.Fee.Equal(tC.expectedPenalty))
			})
		}
	}
}

func TestSLAPerformanceMultiEpochFeePenalty(t *testing.T) {
	testCases := []struct {
		desc            string
		epochsOffBook   int
		epochsOnBook    int
		startWithOnBook bool
		expectedPenalty num.Decimal
	}{
		{
			desc:            "Selects average hysteresis period penalty (3 epochs) over lower current penalty, 0042-LIQF-039",
			epochsOffBook:   3,
			epochsOnBook:    1,
			expectedPenalty: num.DecimalFromFloat(0.75),
		},
		{
			desc:            "Selects average hysteresis period penalty (2 epochs) of 0.5 over 2 epochs, 0042-LIQF-039",
			epochsOffBook:   2,
			epochsOnBook:    2,
			expectedPenalty: num.DecimalFromFloat(0.5),
		},
		{
			desc:            "Selects current higher penalty over hysteresis average period, 0042-LIQF-040",
			epochsOnBook:    4,
			startWithOnBook: true,
			epochsOffBook:   1,
			expectedPenalty: num.DecimalFromFloat(1),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			te := newTestEngine(t)

			// set the net params
			te.engine.OnPerformanceHysteresisEpochsUpdate(4)

			idGen := &stubIDGen{}
			ctx := context.Background()

			te.broker.EXPECT().Send(gomock.Any()).AnyTimes()

			lps := &types.LiquidityProvisionSubmission{
				MarketID:         te.marketID,
				CommitmentAmount: num.NewUint(100),
				Fee:              num.NewDecimalFromFloat(0.5),
				Reference:        fmt.Sprintf("provision-by-%s", partyID),
			}

			err := te.engine.SubmitLiquidityProvision(ctx, lps, partyID, idGen)
			require.NoError(t, err)

			te.auctionState.EXPECT().InAuction().Return(false).AnyTimes()

			te.orderbook.EXPECT().GetLastTradedPrice().Return(num.NewUint(15)).AnyTimes()
			te.orderbook.EXPECT().GetIndicativePrice().Return(num.NewUint(15)).AnyTimes()

			te.orderbook.EXPECT().GetBestStaticBidPrice().Return(num.NewUint(20), nil).AnyTimes()
			te.orderbook.EXPECT().GetBestStaticAskPrice().Return(num.NewUint(10), nil).AnyTimes()
			orders := []*types.Order{}
			te.orderbook.EXPECT().GetOrdersPerParty(partyID).DoAndReturn(func(party string) []*types.Order {
				return orders
			}).AnyTimes()

			epochLength := time.Duration(4) * time.Second
			epochStart := time.Now().Add(-epochLength)
			epochEnd := epochStart.Add(epochLength)

			txs := []liquidity.TX{{ID: "1"}, {ID: "2"}, {ID: "3"}}
			k := te.engine.GenerateKSla(txs)

			firstEpochIters := tC.epochsOffBook
			secondEpochIters := tC.epochsOnBook

			if tC.startWithOnBook {
				orders = generateOrders(*idGen, te.marketID, []uint64{15, 15, 17, 18}, []uint64{12, 12, 12})
				firstEpochIters = tC.epochsOnBook
				secondEpochIters = tC.epochsOffBook
			}

			for i := 0; i < firstEpochIters; i++ {
				te.engine.ResetSLAEpoch(epochStart)

				for j := 0; j < int(epochLength.Seconds()); j++ {
					te.tsvc.SetTime(epochStart.Add(time.Duration(j) * time.Second))
					te.engine.BeginBlock(txs)
					te.engine.TxProcessed(k)
				}

				te.engine.CalculateSLAPenalties(epochEnd)
			}

			if tC.startWithOnBook {
				orders = []*types.Order{}
			} else {
				orders = generateOrders(*idGen, te.marketID, []uint64{15, 15, 17, 18}, []uint64{12, 12, 12})
			}

			for i := 0; i < secondEpochIters; i++ {
				te.engine.ResetSLAEpoch(epochStart)

				for j := 0; j < int(epochLength.Seconds()); j++ {
					te.tsvc.SetTime(epochStart.Add(time.Duration(j) * time.Second))
					te.engine.BeginBlock(txs)
					te.engine.TxProcessed(k)
				}

				te.engine.CalculateSLAPenalties(epochEnd)
			}

			te.engine.CalculateSLAPenalties(epochEnd)
			sla := te.engine.GetSLAPenalties()[partyID]

			fmt.Printf("actual penalty: %s, expected penalty: %s \n", sla.Fee, tC.expectedPenalty)
			require.True(t, sla.Fee.Equal(tC.expectedPenalty))
		})
	}
}

func TestSLAPerformanceBondPenalty(t *testing.T) {
	testCases := []struct {
		desc string

		// represents list of active orders by a party on a book in a given block
		buyOrdersPerBlock   [][]uint64
		sellsOrdersPerBlock [][]uint64

		epochLength int

		// optional net params to set
		commitmentMinTimeFraction      *num.Decimal
		nonPerformanceBondPenaltySlope *num.Decimal
		nonPerformanceBondPenaltyMax   *num.Decimal

		// expected result
		expectedPenalty num.Decimal
	}{
		{
			desc:                      "Bond account penalty is 0 when commitment is met, 0044-LIME-013",
			epochLength:               3,
			buyOrdersPerBlock:         [][]uint64{{15, 15, 17, 18}, {15, 15, 17, 18}, {15, 15, 17, 18}},
			sellsOrdersPerBlock:       [][]uint64{{12, 12, 12}, {12, 12, 12}, {12, 12, 12}},
			commitmentMinTimeFraction: toPoint(num.NewDecimalFromFloat(0.6)),
			expectedPenalty:           num.DecimalFromFloat(0),
		},
		{
			desc:        "Bond account penalty is 35%, 0044-LIME-014",
			epochLength: 10,
			buyOrdersPerBlock: [][]uint64{
				{}, {}, {15, 15, 17, 18}, {15, 15, 17, 18}, {15, 15, 17, 18}, {}, {}, {}, {}, {},
			},
			sellsOrdersPerBlock: [][]uint64{
				{}, {}, {12, 12, 12}, {12, 12, 12}, {12, 12, 12}, {}, {}, {}, {}, {},
			},
			commitmentMinTimeFraction:      toPoint(num.NewDecimalFromFloat(0.6)),
			nonPerformanceBondPenaltySlope: toPoint(num.NewDecimalFromFloat(0.7)),
			nonPerformanceBondPenaltyMax:   toPoint(num.NewDecimalFromFloat(0.6)),
			expectedPenalty:                num.DecimalFromFloat(0.35),
		},
		{
			desc:                           "Bond account penalty is 60%, 0044-LIME-015",
			epochLength:                    3,
			buyOrdersPerBlock:              [][]uint64{{}, {}, {}},
			sellsOrdersPerBlock:            [][]uint64{{}, {}, {}},
			commitmentMinTimeFraction:      toPoint(num.NewDecimalFromFloat(0.6)),
			nonPerformanceBondPenaltySlope: toPoint(num.NewDecimalFromFloat(0.7)),
			nonPerformanceBondPenaltyMax:   toPoint(num.NewDecimalFromFloat(0.6)),
			expectedPenalty:                num.DecimalFromFloat(0.6),
		},
		{
			desc:                           "Bond account penalty is 20%, 0044-LIME-016",
			epochLength:                    3,
			buyOrdersPerBlock:              [][]uint64{{}, {}, {}},
			sellsOrdersPerBlock:            [][]uint64{{}, {}, {}},
			commitmentMinTimeFraction:      toPoint(num.NewDecimalFromFloat(0.6)),
			nonPerformanceBondPenaltySlope: toPoint(num.NewDecimalFromFloat(0.2)),
			nonPerformanceBondPenaltyMax:   toPoint(num.NewDecimalFromFloat(0.6)),
			expectedPenalty:                num.DecimalFromFloat(0.2),
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			te := newTestEngine(t)

			// set the net params
			if tC.commitmentMinTimeFraction != nil {
				te.engine.OnCommitmentMinTimeFractionUpdate(*tC.commitmentMinTimeFraction)
			}
			if tC.nonPerformanceBondPenaltySlope != nil {
				te.engine.OnNonPerformanceBondPenaltySlopeUpdate(*tC.nonPerformanceBondPenaltySlope)
			}
			if tC.nonPerformanceBondPenaltyMax != nil {
				te.engine.OnNonPerformanceBondPenaltyMaxUpdate(*tC.nonPerformanceBondPenaltyMax)
			}

			idGen := &stubIDGen{}
			ctx := context.Background()
			party := "lp-party-1"

			te.broker.EXPECT().Send(gomock.Any()).AnyTimes()

			lps := &types.LiquidityProvisionSubmission{
				MarketID:         te.marketID,
				CommitmentAmount: num.NewUint(100),
				Fee:              num.NewDecimalFromFloat(0.5),
				Reference:        fmt.Sprintf("provision-by-%s", party),
			}

			err := te.engine.SubmitLiquidityProvision(ctx, lps, party, idGen)
			require.NoError(t, err)

			te.auctionState.EXPECT().InAuction().Return(false).AnyTimes()

			te.orderbook.EXPECT().GetLastTradedPrice().Return(num.NewUint(15)).AnyTimes()
			te.orderbook.EXPECT().GetIndicativePrice().Return(num.NewUint(15)).AnyTimes()

			te.orderbook.EXPECT().GetBestStaticBidPrice().Return(num.NewUint(20), nil).AnyTimes()
			te.orderbook.EXPECT().GetBestStaticAskPrice().Return(num.NewUint(10), nil).AnyTimes()
			orders := []*types.Order{}
			te.orderbook.EXPECT().GetOrdersPerParty(party).DoAndReturn(func(party string) []*types.Order {
				return orders
			}).AnyTimes()

			epochLength := time.Duration(tC.epochLength) * time.Second
			epochStart := time.Now().Add(-epochLength)
			epochEnd := epochStart.Add(epochLength)

			orders = generateOrders(*idGen, te.marketID, tC.buyOrdersPerBlock[0], tC.sellsOrdersPerBlock[0])

			te.engine.ResetSLAEpoch(epochStart)
			txs := []liquidity.TX{{ID: "1"}, {ID: "2"}, {ID: "3"}}

			k := te.engine.GenerateKSla(txs)

			for i := 0; i < tC.epochLength; i++ {
				orders = generateOrders(*idGen, te.marketID, tC.buyOrdersPerBlock[i], tC.sellsOrdersPerBlock[i])

				te.tsvc.SetTime(epochStart.Add(time.Duration(i) * time.Second))
				te.engine.BeginBlock(txs)
				te.engine.TxProcessed(k)
			}

			te.engine.CalculateSLAPenalties(epochEnd)

			sla := te.engine.GetSLAPenalties()[party]

			fmt.Printf("actual penalty: %s, expected penalty: %s \n", sla.Bond, tC.expectedPenalty)
			require.True(t, sla.Bond.Equal(tC.expectedPenalty))
		})
	}
}
