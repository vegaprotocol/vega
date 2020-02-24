// +build !race ignore

package plugins_test

import (
	"testing"

	"code.vegaprotocol.io/vega/events"
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
	testcases := []struct {
		run    string
		pos    posStub
		expect expect
	}{
		{
			run: "Long gets more long",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  100,
						price: 50,
					},
					tradeStub{
						size:  25,
						price: 100,
					},
				},
			},
			expect: expect{
				AverageEntryPrice: 60,
				OpenVolume:        125,
				UnrealisedPNL:     5000,
				RealisedPNL:       0,
			},
		},
		{
			run: "Long gets less long",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  100,
						price: 50,
					},
					tradeStub{
						size:  -25,
						price: 100,
					},
				},
			},
			expect: expect{
				AverageEntryPrice: 50,
				OpenVolume:        75,
				UnrealisedPNL:     3750,
				RealisedPNL:       1250,
			},
		},
		{
			run: "Long gets closed",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  100,
						price: 50,
					},
					tradeStub{
						size:  -100,
						price: 100,
					},
				},
			},
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       5000,
			},
		},
		{
			run: "Long gets turned short",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  100,
						price: 50,
					},
					tradeStub{
						size:  -125,
						price: 100,
					},
				},
			},
			expect: expect{
				OpenVolume:        -25,
				AverageEntryPrice: 100,
				UnrealisedPNL:     0,
				RealisedPNL:       5000,
			},
		},
		{
			run: "Short gets more short",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  -100,
						price: 50,
					},
					tradeStub{
						size:  -25,
						price: 100,
					},
				},
			},
			expect: expect{
				OpenVolume:        -125,
				AverageEntryPrice: 60,
				UnrealisedPNL:     -5000,
				RealisedPNL:       0,
			},
		},
		{
			run: "short gets less short",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  -100,
						price: 50,
					},
					tradeStub{
						size:  25,
						price: 100,
					},
				},
			},
			expect: expect{
				OpenVolume:        -75,
				AverageEntryPrice: 50,
				UnrealisedPNL:     -3750,
				RealisedPNL:       -1250,
			},
		},
		{
			run: "Short gets closed",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  -100,
						price: 50,
					},
					tradeStub{
						size:  100,
						price: 100,
					},
				},
			},
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       -5000,
			},
		},
		{
			run: "Short gets turned long",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 100,
				trades: []events.TradeSettlement{
					tradeStub{
						size:  -100,
						price: 50,
					},
					tradeStub{
						size:  125,
						price: 100,
					},
				},
			},
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: 100,
				UnrealisedPNL:     0,
				RealisedPNL:       -5000,
			},
		},
		{
			run: "Long trade up and down",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 75,
				trades: []events.TradeSettlement{
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
				},
			},
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: 80,
				UnrealisedPNL:     -125,
				RealisedPNL:       -2375,
			},
		},
		{
			run: "Profit before and after turning (start long)",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 25,
				trades: []events.TradeSettlement{
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
				},
			},
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       8750,
			},
		},
		{
			run: "Profit before and after turning (start short)",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 50,
				trades: []events.TradeSettlement{
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
				},
			},
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       8750,
			},
		},
		{
			run: "Profit before and loss after turning (start long)",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 250,
				trades: []events.TradeSettlement{
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
				},
			},
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       -2500,
			},
		},
		{
			run: "Profit before and loss after turning (start short)",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 25,
				trades: []events.TradeSettlement{
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
				},
			},
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: 0,
				UnrealisedPNL:     0,
				RealisedPNL:       3750,
			},
		},
		{
			run: "Scenario from Tamlyn's spreadsheet on Google Drive at https://drive.google.com/open?id=1XJESwh5cypALqlYludWobAOEH1Pz-1xS",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 1010,
				trades: []events.TradeSettlement{
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
				},
			},
			expect: expect{
				OpenVolume:        2,
				AverageEntryPrice: 1010,
				UnrealisedPNL:     0,
				RealisedPNL:       -446,
			},
		},
		{
			run: "Scenario from jeremy",
			pos: posStub{
				mID:   market,
				party: "trader1",
				size:  -2,
				price: 1926,
				trades: []events.TradeSettlement{
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
				},
			},
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
			ch := make(chan []events.SettlePosition)
			ref := 1
			lsch := make(chan []events.LossSocialization)
			position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
			position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
			position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
				if ch != nil {
					close(ch)
					ch = nil
				}
			})
			position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
				if lsch != nil {
					close(lsch)
					lsch = nil
				}
			})
			position.Start(position.ctx)
			ch <- []events.SettlePosition{ps}
			// ensure the settleposition was consumed and processed, by pushing an empty slice
			// this is blocking, and will only unblock after the slice above is consumed
			ch <- []events.SettlePosition{}
			// though we're not using this, let's make sure this channel has seen some "action", too
			// this ensures that the consume loop has seen 2 iterations prior to us making the call
			// to GetPositionsByMarket, data is certain to be up to date
			lsch <- []events.LossSocialization{}
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
