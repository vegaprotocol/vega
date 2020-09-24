package plugins_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"
	"github.com/stretchr/testify/assert"
)

type expect struct {
	AverageEntryPrice int
	OpenVolume        int
	RealisedPNL       int
	UnrealisedPNL     int
}

func TestPositionSpecSuite(t *testing.T) {
	market := "market-id"
	ctx := context.Background()
	testcases := []struct {
		run    string
		pos    plugins.SPE
		expect expect
	}{
		{
			run: "Long gets more long",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 50,
				},
				tradeStub{
					size:  25,
					price: 100,
				},
			}, 1),
			expect: expect{
				AverageEntryPrice: 60,
				OpenVolume:        125,
				UnrealisedPNL:     5000,
				RealisedPNL:       0,
			},
		},
		{
			run: "Long gets less long",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 50,
				},
				tradeStub{
					size:  -25,
					price: 100,
				},
			}, 1),
			expect: expect{
				AverageEntryPrice: 50,
				OpenVolume:        75,
				UnrealisedPNL:     3750,
				RealisedPNL:       1250,
			},
		},
		{
			run: "Long gets closed",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 50,
				},
				tradeStub{
					size:  -100,
					price: 100,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       5000,
			},
		},
		{
			run: "Long gets turned short",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 50,
				},
				tradeStub{
					size:  -125,
					price: 100,
				},
			}, 1),
			expect: expect{
				OpenVolume:        -25,
				AverageEntryPrice: 100,
				UnrealisedPNL:     0,
				RealisedPNL:       5000,
			},
		},
		{
			run: "Short gets more short",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: 50,
				},
				tradeStub{
					size:  -25,
					price: 100,
				},
			}, 1),
			expect: expect{
				OpenVolume:        -125,
				AverageEntryPrice: 60,
				UnrealisedPNL:     -5000,
				RealisedPNL:       0,
			},
		},
		{
			run: "short gets less short",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: 50,
				},
				tradeStub{
					size:  25,
					price: 100,
				},
			}, 1),
			expect: expect{
				OpenVolume:        -75,
				AverageEntryPrice: 50,
				UnrealisedPNL:     -3750,
				RealisedPNL:       -1250,
			},
		},
		{
			run: "Short gets closed",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: 50,
				},
				tradeStub{
					size:  100,
					price: 100,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       -5000,
			},
		},
		{
			run: "Short gets turned long",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: 50,
				},
				tradeStub{
					size:  125,
					price: 100,
				},
			}, 1),
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: 100,
				UnrealisedPNL:     0,
				RealisedPNL:       -5000,
			},
		},
		{
			run: "Long trade up and down",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 75, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 100,
				},
				tradeStub{
					size:  -25,
					price: 25,
				},
				tradeStub{
					size:  50,
					price: 50,
				},
				tradeStub{
					size:  -100,
					price: 75,
				},
			}, 1),
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: 80,
				UnrealisedPNL:     -125,
				RealisedPNL:       -2375,
			},
		},
		{
			run: "Profit before and after turning (start long)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 50,
				},
				tradeStub{
					size:  -150,
					price: 100,
				},
				tradeStub{
					size:  50,
					price: 25,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       8750,
			},
		},
		{
			run: "Profit before and after turning (start short)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: 100,
				},
				tradeStub{
					size:  150,
					price: 25,
				},
				tradeStub{
					size:  -50,
					price: 50,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       8750,
			},
		},
		{
			run: "Profit before and loss after turning (start long)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: 50,
				},
				tradeStub{
					size:  -150,
					price: 100,
				},
				tradeStub{
					size:  50,
					price: 250,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       -2500,
			},
		},
		{
			run: "Profit before and loss after turning (start short)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: 100,
				},
				tradeStub{
					size:  150,
					price: 50,
				},
				tradeStub{
					size:  -50,
					price: 25,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       3750,
			},
		},
		{
			run: "Scenario from Tamlyn's spreadsheet on Google Drive at https://drive.google.com/open?id=1XJESwh5cypALqlYludWobAOEH1Pz-1xS",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 1010, []events.TradeSettlement{
				tradeStub{
					size:  5,
					price: 1000,
				},
				tradeStub{
					size:  2,
					price: 1050,
				},
				tradeStub{
					size:  -4,
					price: 900,
				},
				tradeStub{
					size:  -3,
					price: 1070,
				},
				tradeStub{
					size:  3,
					price: 1060,
				},
				tradeStub{
					size:  -5,
					price: 1010,
				},
				tradeStub{
					size:  -3,
					price: 980,
				},
				tradeStub{
					size:  2,
					price: 1030,
				},
				tradeStub{
					size:  3,
					price: 982,
				},
				tradeStub{
					size:  -4,
					price: 1020,
				},
				tradeStub{
					size:  6,
					price: 1010,
				},
			}, 1),
			expect: expect{
				OpenVolume:        2,
				AverageEntryPrice: 1010,
				UnrealisedPNL:     0,
				RealisedPNL:       -446,
			},
		},
		{
			run: "Scenario from jeremy",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, 100, []events.TradeSettlement{
				tradeStub{
					size:  1,
					price: 1931,
				},
				tradeStub{
					size:  4,
					price: 1931,
				},
				tradeStub{
					size:  -1,
					price: 1923,
				},
				tradeStub{
					size:  -4,
					price: 1923,
				},
				tradeStub{
					size:  7,
					price: 1927,
				},
				tradeStub{
					size:  -2,
					price: 1926,
				},
				tradeStub{
					size:  -1,
					price: 1926,
				},
				tradeStub{
					size:  -4,
					price: 1926,
				},
				tradeStub{
					size:  1,
					price: 1934,
				},
				tradeStub{
					size:  7,
					price: 1933,
				},
				tradeStub{
					size:  1,
					price: 1932,
				},
				tradeStub{
					size:  1,
					price: 1932,
				},
				tradeStub{
					size:  -8,
					price: 1926,
				},
				tradeStub{
					size:  -2,
					price: 1926,
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       -116,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.run, func(t *testing.T) {
			ps := tc.pos
			position := getPosPlugin(t)
			defer position.Finish()
			position.Push(ps)
			pp, err := position.GetPositionsByMarket(market)
			assert.NoError(t, err)
			assert.NotZero(t, len(pp))
			// average entry price should be 1k
			assert.Equal(t, tc.expect.AverageEntryPrice, int(pp[0].AverageEntryPrice), "invalid average entry price")
			assert.Equal(t, tc.expect.OpenVolume, int(pp[0].OpenVolume), "invalid open volume")
			assert.Equal(t, tc.expect.UnrealisedPNL, int(pp[0].UnrealisedPNL), "invalid unrealised pnl")
			assert.Equal(t, tc.expect.RealisedPNL, int(pp[0].RealisedPNL), "invalid realised pnl")
		})
	}
}
