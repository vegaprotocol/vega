// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package service_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/service/mocks"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func getService(t *testing.T) *MDS {
	t.Helper()
	cfg := service.MarketDepthConfig{
		AmmFullExpansionPercentage: 1,
		AmmMaxEstimatedSteps:       5,
		AmmEstimatedStepPercentage: 0.2,
	}
	return getServiceWithConfig(t, cfg)
}

func getServiceWithConfig(t *testing.T, cfg service.MarketDepthConfig) *MDS {
	t.Helper()
	ctrl := gomock.NewController(t)
	pos := mocks.NewMockPositionStore(ctrl)
	orders := mocks.NewMockOrderStore(ctrl)
	marketData := mocks.NewMockMarketDataStore(ctrl)
	amm := mocks.NewMockAMMStore(ctrl)
	markets := mocks.NewMockMarketStore(ctrl)
	assets := mocks.NewMockAssetStore(ctrl)

	return &MDS{
		service:    service.NewMarketDepth(cfg, orders, amm, marketData, pos, assets, markets, logging.NewTestLogger()),
		ctrl:       ctrl,
		pos:        pos,
		amm:        amm,
		orders:     orders,
		marketData: marketData,
		markets:    markets,
		assets:     assets,
	}
}

func TestAMMMarketDepth(t *testing.T) {
	ctx := context.Background()
	mds := getService(t)
	defer mds.ctrl.Finish()

	marketID := vgcrypto.RandomHash()

	ensureLiveOrders(t, mds, marketID)
	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)

	// data node is starting from network history, initialise market-depth based on whats aleady there
	pool := ensureAMMs(t, mds, marketID)
	mds.service.Initialise(ctx)

	assert.Equal(t, 240, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, true)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, false)))
	assert.Equal(t, 260, int(mds.service.GetTotalVolume(marketID)))

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(marketID).String())

	// now pretend that something traded with the AMM and its position is now 10 long
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 10}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)
	mds.service.AddOrder(
		&types.Order{
			ID:       vgcrypto.RandomHash(),
			Party:    pool.AmmPartyID.String(),
			MarketID: marketID,
			Side:     types.SideBuy,
			Status:   entities.OrderStatusFilled,
		},
		time.Date(2022, 3, 8, 16, 15, 39, 901022000, time.UTC),
		37,
	)

	// volume should be the same but the buys and sells should have shifted
	assert.Equal(t, 240, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, true)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, false)))
	assert.Equal(t, 260, int(mds.service.GetTotalVolume(marketID)))

	assert.Equal(t, "1996", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "1998", mds.service.GetBestAskPrice(marketID).String())

	// now the AMM is updated so that its definition has changed, namely that its curve when short is removed
	pool.ParametersUpperBound = nil
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 10}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)
	mds.service.OnAMMUpdate(pool, time.Now(), 999)

	// volume should change
	assert.Equal(t, 125, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 65, int(mds.service.GetAMMVolume(marketID, true)))
	assert.Equal(t, 60, int(mds.service.GetAMMVolume(marketID, false)))
	assert.Equal(t, 145, int(mds.service.GetTotalVolume(marketID)))
	assert.Equal(t, "1996", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "1998", mds.service.GetBestAskPrice(marketID).String())

	// and there should definitely be no volume at 2001
	assert.Equal(t, 0, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2001)))

	// now the AMM is cancelled, we expect all AMM volume to be removed
	pool.Status = entities.AMMStatusCancelled
	mds.service.OnAMMUpdate(pool, time.Now(), 1000)

	assert.Equal(t, 0, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 20, int(mds.service.GetTotalVolume(marketID)))
}

func TestAMMInitialiseNoAMM(t *testing.T) {
	ctx := context.Background()
	mds := getService(t)
	defer mds.ctrl.Finish()

	marketID := vgcrypto.RandomHash()

	ensureLiveOrders(t, mds, marketID)

	// initialise when there are no AMMs
	mds.amm.EXPECT().ListActive(gomock.Any()).Return(nil, nil).Times(1)
	mds.service.Initialise(ctx)
	assert.Equal(t, 0, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 20, int(mds.service.GetTotalVolume(marketID)))

	// now a new AMM for a new market appears
	newMarket := vgcrypto.RandomHash()
	pool := getAMMDefinition(t, newMarket)

	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)
	mds.service.OnAMMUpdate(pool, time.Now(), 1000)

	// check it makes sense
	assert.Equal(t, 240, int(mds.service.GetTotalAMMVolume(newMarket)))
	assert.Equal(t, "1999", mds.service.GetBestBidPrice(newMarket).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(newMarket).String())
}

func TestAMMStepOverFairPrice(t *testing.T) {
	ctx := context.Background()
	mds := getService(t)
	defer mds.ctrl.Finish()

	// this is for an awkward case where an AMM's position exists between the position of two ticks
	// for example if an AMM's base is at 2000, and it has 5 volume between 2000 -> 2001 our accurate
	// expansion will step from 2000 -> 2001 and say there is 5 SELL volume at price 2001.
	//
	// Say the AMM now trades 1, when we expand and step from 2000 -> 2001 there should be only 4 SELL volume
	// at 2001 and now 1 BUY volume at 1999

	marketID := vgcrypto.RandomHash()
	ensureLiveOrders(t, mds, marketID)
	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)

	// data node is starting from network history, initialise market-depth based on whats aleady there
	pool := ensureAMMs(t, mds, marketID)
	mds.service.Initialise(ctx)

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(marketID).String())
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideBuy, 1999)))
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2001)))

	// now a single trade happens making the AMM 1 short
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 1}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)
	mds.service.AddOrder(
		&types.Order{
			ID:       vgcrypto.RandomHash(),
			Party:    pool.AmmPartyID.String(),
			MarketID: marketID,
			Side:     types.SideBuy,
			Status:   entities.OrderStatusFilled,
		},
		time.Date(2022, 3, 8, 16, 15, 39, 901022000, time.UTC),
		37,
	)

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2000", mds.service.GetBestAskPrice(marketID).String())
	assert.Equal(t, 2, int(mds.service.GetVolumeAtPrice(marketID, types.SideBuy, 1999)))
	assert.Equal(t, 1, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2000)))
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2001)))
}

func TestAMMSmallBounds(t *testing.T) {
	ctx := context.Background()
	mds := getServiceWithConfig(t,
		service.MarketDepthConfig{
			AmmFullExpansionPercentage: 0.000001,
			AmmEstimatedStepPercentage: 0.000001,
			AmmMaxEstimatedSteps:       2,
		},
	)
	defer mds.ctrl.Finish()

	marketID := vgcrypto.RandomHash()
	ensureLiveOrders(t, mds, marketID)
	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)

	// data node is starting from network history, initialise market-depth based on whats aleady there
	ensureAMMs(t, mds, marketID)
	mds.service.Initialise(ctx)

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(marketID).String())
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideBuy, 1999)))
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2001)))

	// anywhere else is zero
	assert.Equal(t, 0, int(mds.service.GetVolumeAtPrice(marketID, types.SideBuy, 1998)))
	assert.Equal(t, 0, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2002)))
}

func TestEstimatedStepOverAMMBound(t *testing.T) {
	ctx := context.Background()
	mds := getServiceWithConfig(t,
		service.MarketDepthConfig{
			AmmFullExpansionPercentage: 5,
			AmmEstimatedStepPercentage: 7.6, // make this a werid number so our estimated steps are not nice multiplies of 10
			AmmMaxEstimatedSteps:       5,
		},
	)
	defer mds.ctrl.Finish()

	marketID := vgcrypto.RandomHash()
	ensureLiveOrders(t, mds, marketID)
	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)

	// data node is starting from network history, initialise market-depth based on whats aleady there
	ensureAMMs(t, mds, marketID)
	mds.service.Initialise(ctx)

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(marketID).String())
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideBuy, 1999)))
	assert.Equal(t, 3, int(mds.service.GetVolumeAtPrice(marketID, types.SideSell, 2001)))
}

func TestExpansionMuchBiggerThanAMMs(t *testing.T) {
	ctx := context.Background()

	cfg := service.MarketDepthConfig{
		AmmFullExpansionPercentage: 1,
		AmmMaxEstimatedSteps:       10,
		AmmEstimatedStepPercentage: 5,
	}

	mds := getServiceWithConfig(t, cfg)
	defer mds.ctrl.Finish()

	marketID := vgcrypto.RandomHash()

	ensureLiveOrders(t, mds, marketID)
	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)

	// data node is starting from network history, initialise market-depth based on whats aleady there
	ensureAMMs(t, mds, marketID)
	mds.service.Initialise(ctx)

	assert.Equal(t, 465, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 345, int(mds.service.GetAMMVolume(marketID, true)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, false)))
	assert.Equal(t, 485, int(mds.service.GetTotalVolume(marketID)))

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(marketID).String())
}

func TestMidPriceMove(t *testing.T) {
	ctx := context.Background()

	mds := getService(t)
	defer mds.ctrl.Finish()

	marketID := vgcrypto.RandomHash()

	ensureLiveOrders(t, mds, marketID)
	ensureDecimalPlaces(t, mds)
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 0}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(2000)}, nil)

	// data node is starting from network history, initialise market-depth based on whats aleady there
	pool := ensureAMMs(t, mds, marketID)
	mds.service.Initialise(ctx)

	assert.Equal(t, 240, int(mds.service.GetTotalAMMVolume(marketID)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, true)))
	assert.Equal(t, 120, int(mds.service.GetAMMVolume(marketID, false)))
	assert.Equal(t, 260, int(mds.service.GetTotalVolume(marketID)))

	assert.Equal(t, "1999", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "2001", mds.service.GetBestAskPrice(marketID).String())

	// now say the mid-price moves a little, we want to check we recalculate the levels properly
	mds.pos.EXPECT().GetByMarketAndParty(gomock.Any(), gomock.Any(), gomock.Any()).Return(entities.Position{OpenVolume: 500}, nil)
	mds.marketData.EXPECT().GetMarketDataByID(gomock.Any(), gomock.Any()).Times(1).Return(entities.MarketData{MidPrice: num.DecimalFromInt64(1800)}, nil)
	mds.service.AddOrder(
		&types.Order{
			ID:       vgcrypto.RandomHash(),
			Party:    pool.AmmPartyID.String(),
			MarketID: marketID,
			Side:     types.SideBuy,
			Status:   entities.OrderStatusFilled,
		},
		time.Date(2022, 3, 8, 16, 15, 39, 901022000, time.UTC),
		37,
	)

	assert.Equal(t, "1828", mds.service.GetBestBidPrice(marketID).String())
	assert.Equal(t, "3000", mds.service.GetBestAskPrice(marketID).String()) // this is an actual order volume not AMM volume
}

func ensureLiveOrders(t *testing.T, mds *MDS, marketID string) {
	t.Helper()
	mds.orders.EXPECT().GetLiveOrders(gomock.Any()).Return([]entities.Order{
		{
			ID:        entities.OrderID(vgcrypto.RandomHash()),
			MarketID:  entities.MarketID(marketID),
			PartyID:   entities.PartyID(vgcrypto.RandomHash()),
			Side:      types.SideBuy,
			Price:     decimal.NewFromInt(1000),
			Size:      10,
			Remaining: 10,
			Type:      entities.OrderTypeLimit,
			Status:    entities.OrderStatusActive,
			VegaTime:  time.Date(2022, 3, 8, 14, 14, 45, 762739000, time.UTC),
			SeqNum:    32,
		},
		{
			ID:        entities.OrderID(vgcrypto.RandomHash()),
			MarketID:  entities.MarketID(marketID),
			PartyID:   entities.PartyID(vgcrypto.RandomHash()),
			Side:      types.SideSell,
			Type:      entities.OrderTypeLimit,
			Status:    entities.OrderStatusActive,
			Price:     decimal.NewFromInt(3000),
			Size:      10,
			Remaining: 10,
			VegaTime:  time.Date(2022, 3, 8, 14, 15, 39, 901022000, time.UTC),
			SeqNum:    33,
		},
	}, nil).Times(1)
}

func getAMMDefinition(t *testing.T, marketID string) entities.AMMPool {
	t.Helper()
	return entities.AMMPool{
		PartyID:                  entities.PartyID(vgcrypto.RandomHash()),
		AmmPartyID:               entities.PartyID(vgcrypto.RandomHash()),
		MarketID:                 entities.MarketID(marketID),
		ParametersLowerBound:     ptr.From(num.DecimalFromInt64(1800)),
		LowerVirtualLiquidity:    num.DecimalFromFloat(580723.51752738390596462639919437474617),
		LowerTheoreticalPosition: num.DecimalFromFloat(702.4119613637248987),
		ParametersBase:           num.DecimalFromInt64(2000),
		ParametersUpperBound:     ptr.From(num.DecimalFromInt64(2200)),
		UpperVirtualLiquidity:    num.DecimalFromFloat(610600.1174758454383959875699679680084),
		UpperTheoreticalPosition: num.DecimalFromFloat(635.3954521864637116),
	}
}

func ensureAMMs(t *testing.T, mds *MDS, marketID string) entities.AMMPool {
	t.Helper()

	pool := getAMMDefinition(t, marketID)
	mds.amm.EXPECT().ListActive(gomock.Any()).Return([]entities.AMMPool{pool}, nil).Times(1)
	return pool
}

func ensureDecimalPlaces(t *testing.T, mds *MDS) {
	t.Helper()

	market := entities.Market{
		TradableInstrument: entities.TradableInstrument{
			TradableInstrument: &vega.TradableInstrument{
				Instrument: &vega.Instrument{
					Product: &vega.Instrument_Future{
						Future: &vega.Future{},
					},
				},
			},
		},
		DecimalPlaces: 1,
		TickSize:      ptr.From(num.DecimalOne()),
	}
	mds.markets.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(market, nil)

	asset := entities.Asset{
		Decimals: 1,
	}
	mds.assets.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(asset, nil)
}
