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

package sqlsubscribers_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable.
import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/plugins"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
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
		pos    plugins.SPE // TODO
		expect expect
	}{
		{
			run: "Long gets more long",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  25,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				AverageEntryPrice: num.NewUint(60),
				OpenVolume:        125,
				UnrealisedPNL:     num.NewDecimalFromFloat(5000.0),
				RealisedPNL:       num.NewDecimalFromFloat(0.0),
			},
		},
		{
			run: "Long gets less long",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -25,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				AverageEntryPrice: num.NewUint(50),
				OpenVolume:        75,
				UnrealisedPNL:     num.NewDecimalFromFloat(3750),
				RealisedPNL:       num.NewDecimalFromFloat(1250),
			},
		},
		{
			run: "Long gets closed",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -100,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(5000),
			},
		},
		{
			run: "Long gets turned short",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -125,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        -25,
				AverageEntryPrice: num.NewUint(100),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(5000),
			},
		},
		{
			run: "Short gets more short",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  -25,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        -125,
				AverageEntryPrice: num.NewUint(60),
				UnrealisedPNL:     num.NewDecimalFromFloat(-5000),
				RealisedPNL:       num.NewDecimalFromFloat(0),
			},
		},
		{
			run: "short gets less short",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  25,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        -75,
				AverageEntryPrice: num.NewUint(50),
				UnrealisedPNL:     num.NewDecimalFromFloat(-3750),
				RealisedPNL:       num.NewDecimalFromFloat(-1250),
			},
		},
		{
			run: "Short gets closed",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  100,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-5000),
			},
		},
		{
			run: "Short gets turned long",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
				tradeStub{
					size:  -100,
					price: num.NewUint(50),
				},
				tradeStub{
					size:  125,
					price: num.NewUint(100),
				},
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: num.NewUint(100),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-5000),
			},
		},
		{
			run: "Long trade up and down",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(75), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        25,
				AverageEntryPrice: num.NewUint(80),
				UnrealisedPNL:     num.NewDecimalFromFloat(-125),
				RealisedPNL:       num.NewDecimalFromFloat(-2375),
			},
		},
		{
			run: "Profit before and after turning (start long)",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(8750),
			},
		},
		{
			run: "Profit before and after turning (start short)",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(8750),
			},
		},
		{
			run: "Profit before and loss after turning (start long)",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-2500),
			},
		},
		{
			run: "Profit before and loss after turning (start short)",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(3750),
			},
		},
		{
			run: "Scenario from Tamlyn's spreadsheet on Google Drive at https://drive.google.com/open?id=1XJESwh5cypALqlYludWobAOEH1Pz-1xS",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(1010), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        2,
				AverageEntryPrice: num.NewUint(1010),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-446),
			},
		},
		{
			run: "Scenario from jeremy",
			pos: events.NewSettlePositionEvent(ctx, "party1", market, num.NewUint(100), []events.TradeSettlement{
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
			}, 1, num.DecimalFromFloat(1)),
			expect: expect{
				OpenVolume:        0,
				AverageEntryPrice: num.Zero(),
				UnrealisedPNL:     num.NewDecimalFromFloat(0),
				RealisedPNL:       num.NewDecimalFromFloat(-116),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.run, func(t *testing.T) {
			ps := tc.pos
			sub, store := getSubscriberAndStore(t)
			sub.Push(context.Background(), ps)
			pp, err := store.GetByMarket(ctx, entities.NewMarketID(market))
			assert.NoError(t, err)
			assert.NotZero(t, len(pp))
			// average entry price should be 1k
			assert.Equal(t, tc.expect.AverageEntryPrice, pp[0].AverageEntryPriceUint(), "invalid average entry price")
			assert.Equal(t, tc.expect.OpenVolume, pp[0].OpenVolume, "invalid open volume")
			assert.Equal(t, tc.expect.UnrealisedPNL.String(), pp[0].UnrealisedPnl.Round(0).String(), "invalid unrealised pnl")
			assert.Equal(t, tc.expect.RealisedPNL.String(), pp[0].RealisedPnl.Round(0).String(), "invalid realised pnl")
		})
	}
}

func getSubscriberAndStore(t *testing.T) (*sqlsubscribers.Position, sqlsubscribers.PositionStore) {
	t.Helper()
	ctrl := gomock.NewController(t)

	store := mocks.NewMockPositionStore(ctrl)

	var lastPos entities.Position
	recordPos := func(_ context.Context, pos entities.Position) error {
		lastPos = pos
		return nil
	}

	getByMarket := func(_ context.Context, _ entities.MarketID) ([]entities.Position, error) {
		return []entities.Position{lastPos}, nil
	}

	getByMarketAndParty := func(_ context.Context, _ entities.MarketID, _ entities.PartyID) (entities.Position, error) {
		return lastPos, nil
	}

	store.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(recordPos)
	store.EXPECT().GetByMarket(gomock.Any(), gomock.Any()).DoAndReturn(getByMarket)
	store.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(getByMarketAndParty)

	p := sqlsubscribers.NewPosition(store, logging.NewTestLogger())
	return p, store
}

type tradeStub struct {
	size  int64
	price *num.Uint
}

func (t tradeStub) Size() int64 {
	return t.size
}

func (t tradeStub) Price() *num.Uint {
	return t.price.Clone()
}
