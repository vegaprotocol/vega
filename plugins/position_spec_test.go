package plugins_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

type expect struct {
	AverageEntryPrice *num.Uint
	OpenVolume        int64
	RealisedPNL       num.Decimal
	UnrealisedPNL     num.Decimal
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
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  25,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				AverageEntryPrice: num.NewUint(60),
				OpenVolume:        125,
				UnrealisedPNL:     num.NewDecimalFromFloat(5000.0),
				RealisedPNL:       num.NewDecimalFromFloat(0.0),
			},
		},
		{
			run: "Long gets less long",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -25,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				AverageEntryPrice: num.NewUint(50),
				OpenVolume:        75,
				UnrealisedPNL:     num.NewDecimalFromFloat(3750),
				RealisedPNL:       num.NewDecimalFromFloat(1250),
			},
		},
		{
			run: "Long gets closed",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -100,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(5000),
			},
		},
		{
			run: "Long gets turned short",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -125,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				OpenVolume:        -25,
				AverageEntryPrice: num.NewUint(100),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(5000),
			},
		},
		{
			run: "Short gets more short",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -25,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				OpenVolume:        -125,
				AverageEntryPrice: num.NewUint(60),
				UnrealisedPNL:     num.NewDecimalFromFloat(-5000),
				RealisedPNL:       num.NewDecimalFromFloat(0),
			},
		},
		{
			run: "short gets less short",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  25,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				OpenVolume:        -75,
				AverageEntryPrice: num.NewUint(50),
				UnrealisedPNL:     num.NewDecimalFromFloat(-3750),
				RealisedPNL:       num.NewDecimalFromFloat(-1250),
			},
		},
		{
			run: "Short gets closed",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  100,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-5000),
			},
		},
		{
			run: "Short gets turned long",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  125,
					price: num.NewUint(100),
				},
			}, 1),
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: num.NewUint(100),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-5000),
			},
		},
		{
			run: "Long trade up and down",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(75), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(100),
				},
				tradeStub{
					size:  -25,
					price: num.NewUint(25),
				},
				tradeStub{
					size:  50,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -100,
					price: num.NewUint(75),
				},
			}, 1),
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: num.NewUint(80),
				UnrealisedPNL:     num.NewDecimalFromFloat(-125),
				RealisedPNL:       num.NewDecimalFromFloat(-2375),
			},
		},
		{
			run: "Profit before and after turning (start long)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -150,
					price: num.NewUint(100),
				},
				tradeStub{
					size:  50,
					price: num.NewUint(25),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(8750),
			},
		},
		{
			run: "Profit before and after turning (start short)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(100),
				},
				tradeStub{
					size:  150,
					price: num.NewUint(25),
				},
				tradeStub{
					size:  -50,
					price: num.NewUint(50),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(8750),
			},
		},
		{
			run: "Profit before and loss after turning (start long)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -150,
					price: num.NewUint(100),
				},
				tradeStub{
					size:  50,
					price: num.NewUint(250),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-2500),
			},
		},
		{
			run: "Profit before and loss after turning (start short)",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(100),
				},
				tradeStub{
					size:  150,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -50,
					price: num.NewUint(25),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(3750),
			},
		},
		{
			run: "Scenario from Tamlyn's spreadsheet on Google Drive at https://drive.google.com/open?id=1XJESwh5cypALqlYludWobAOEH1Pz-1xS",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(1010), []events.TradeSettlement{
				tradeStub{
					size:  5,
					price: num.NewUint(1000),
				},
				tradeStub{
					size:  2,
					price: num.NewUint(1050),
				},
				tradeStub{
					size:  -4,
					price: num.NewUint(900),
				},
				tradeStub{
					size:  -3,
					price: num.NewUint(1070),
				},
				tradeStub{
					size:  3,
					price: num.NewUint(1060),
				},
				tradeStub{
					size:  -5,
					price: num.NewUint(1010),
				},
				tradeStub{
					size:  -3,
					price: num.NewUint(980),
				},
				tradeStub{
					size:  2,
					price: num.NewUint(1030),
				},
				tradeStub{
					size:  3,
					price: num.NewUint(982),
				},
				tradeStub{
					size:  -4,
					price: num.NewUint(1020),
				},
				tradeStub{
					size:  6,
					price: num.NewUint(1010),
				},
			}, 1),
			expect: expect{
				OpenVolume:        2,
				AverageEntryPrice: num.NewUint(1010),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-446),
			},
		},
		{
			run: "Scenario from jeremy",
			pos: events.NewSettlePositionEvent(ctx, "trader1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  1,
					price: num.NewUint(1931),
				},
				tradeStub{
					size:  4,
					price: num.NewUint(1931),
				},
				tradeStub{
					size:  -1,
					price: num.NewUint(1923),
				},
				tradeStub{
					size:  -4,
					price: num.NewUint(1923),
				},
				tradeStub{
					size:  7,
					price: num.NewUint(1927),
				},
				tradeStub{
					size:  -2,
					price: num.NewUint(1926),
				},
				tradeStub{
					size:  -1,
					price: num.NewUint(1926),
				},
				tradeStub{
					size:  -4,
					price: num.NewUint(1926),
				},
				tradeStub{
					size:  1,
					price: num.NewUint(1934),
				},
				tradeStub{
					size:  7,
					price: num.NewUint(1933),
				},
				tradeStub{
					size:  1,
					price: num.NewUint(1932),
				},
				tradeStub{
					size:  1,
					price: num.NewUint(1932),
				},
				tradeStub{
					size:  -8,
					price: num.NewUint(1926),
				},
				tradeStub{
					size:  -2,
					price: num.NewUint(1926),
				},
			}, 1),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.NewUint(0),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-116),
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
			assert.Equal(t, tc.expect.AverageEntryPrice, pp[0].AverageEntryPrice, "invalid average entry price")
			assert.Equal(t, tc.expect.OpenVolume, pp[0].OpenVolume, "invalid open volume")
			assert.Equal(t, tc.expect.UnrealisedPNL.String(), pp[0].UnrealisedPnl.String(), "invalid unrealised pnl")
			assert.Equal(t, tc.expect.RealisedPNL.String(), pp[0].RealisedPnl.String(), "invalid realised pnl")
		})
	}
}
