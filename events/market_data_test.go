package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestMarketDataDeepClone(t *testing.T) {
	ctx := context.Background()

	md := proto.MarketData{
		MarkPrice:             1000,
		BestBidPrice:          2000,
		BestBidVolume:         3000,
		BestOfferPrice:        4000,
		BestOfferVolume:       5000,
		BestStaticBidPrice:    6000,
		BestStaticBidVolume:   7000,
		BestStaticOfferPrice:  8000,
		BestStaticOfferVolume: 9000,
		MidPrice:              10000,
		StaticMidPrice:        11000,
		Market:                "Market",
		Timestamp:             12000,
		OpenInterest:          13000,
		AuctionEnd:            14000,
		AuctionStart:          15000,
		IndicativePrice:       16000,
		IndicativeVolume:      17000,
		MarketTradingMode:     proto.Market_TRADING_MODE_CONTINUOUS,
		Trigger:               proto.AuctionTrigger_AUCTION_TRIGGER_OPENING,
		TargetStake:           "18000",
		SuppliedStake:         "19000",
		PriceMonitoringBounds: []*proto.PriceMonitoringBounds{
			&proto.PriceMonitoringBounds{
				MinValidPrice: 20000,
				MaxValidPrice: 21000,
				Trigger: &proto.PriceMonitoringTrigger{
					Horizon:          22000,
					Probability:      123.45,
					AuctionExtension: 23000,
				},
				ReferencePrice: 24000,
			},
		},
		MarketValueProxy: "MVP",
		LiquidityProviderFeeShare: []*proto.LiquidityProviderFeeShare{
			&proto.LiquidityProviderFeeShare{
				Party:                 "Party",
				EquityLikeShare:       "25000",
				AverageEntryValuation: "26000",
			},
		},
	}

	marketDataEvent := events.NewMarketDataEvent(ctx, md)
	md2 := marketDataEvent.MarketData()

	// Change the original and check we are not updating the wrapped event
	md.MarkPrice = 999
	md.BestBidPrice = 999
	md.BestBidVolume = 999
	md.BestOfferPrice = 999
	md.BestOfferVolume = 999
	md.BestStaticBidPrice = 999
	md.BestStaticBidVolume = 999
	md.BestStaticOfferPrice = 999
	md.BestStaticOfferVolume = 999
	md.MidPrice = 999
	md.StaticMidPrice = 999
	md.Market = "Changed"
	md.Timestamp = 999
	md.OpenInterest = 999
	md.AuctionEnd = 999
	md.AuctionStart = 999
	md.IndicativePrice = 999
	md.IndicativeVolume = 999
	md.MarketTradingMode = proto.Market_TRADING_MODE_UNSPECIFIED
	md.Trigger = proto.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED
	md.TargetStake = "999"
	md.SuppliedStake = "999"
	md.PriceMonitoringBounds[0].MinValidPrice = 999
	md.PriceMonitoringBounds[0].MaxValidPrice = 999
	md.PriceMonitoringBounds[0].Trigger.Horizon = 999
	md.PriceMonitoringBounds[0].Trigger.Probability = 999
	md.PriceMonitoringBounds[0].Trigger.AuctionExtension = 999
	md.PriceMonitoringBounds[0].ReferencePrice = 999
	md.MarketValueProxy = "Changed"
	md.LiquidityProviderFeeShare[0].Party = "Changed"
	md.LiquidityProviderFeeShare[0].EquityLikeShare = "999"
	md.LiquidityProviderFeeShare[0].AverageEntryValuation = "999"

	assert.NotEqual(t, md.MarkPrice, md2.MarkPrice)
	assert.NotEqual(t, md.BestBidPrice, md2.BestBidPrice)
	assert.NotEqual(t, md.BestBidVolume, md2.BestBidVolume)
	assert.NotEqual(t, md.BestOfferPrice, md2.BestOfferPrice)
	assert.NotEqual(t, md.BestOfferVolume, md2.BestOfferVolume)
	assert.NotEqual(t, md.BestStaticBidPrice, md2.BestStaticBidPrice)
	assert.NotEqual(t, md.BestStaticBidVolume, md2.BestStaticBidVolume)
	assert.NotEqual(t, md.BestStaticOfferPrice, md2.BestStaticOfferPrice)
	assert.NotEqual(t, md.BestStaticOfferVolume, md2.BestStaticOfferVolume)
	assert.NotEqual(t, md.MidPrice, md2.MidPrice)
	assert.NotEqual(t, md.StaticMidPrice, md2.StaticMidPrice)
	assert.NotEqual(t, md.Market, md2.Market)
	assert.NotEqual(t, md.Timestamp, md2.Timestamp)
	assert.NotEqual(t, md.OpenInterest, md2.OpenInterest)
	assert.NotEqual(t, md.AuctionEnd, md2.AuctionEnd)
	assert.NotEqual(t, md.AuctionStart, md2.AuctionStart)
	assert.NotEqual(t, md.IndicativePrice, md2.IndicativePrice)
	assert.NotEqual(t, md.IndicativeVolume, md2.IndicativeVolume)
	assert.NotEqual(t, md.MarketTradingMode, md2.MarketTradingMode)
	assert.NotEqual(t, md.Trigger, md2.Trigger)
	assert.NotEqual(t, md.TargetStake, md2.TargetStake)
	assert.NotEqual(t, md.SuppliedStake, md2.SuppliedStake)
	assert.NotEqual(t, md.PriceMonitoringBounds[0].MinValidPrice, md2.PriceMonitoringBounds[0].MinValidPrice)
	assert.NotEqual(t, md.PriceMonitoringBounds[0].MaxValidPrice, md2.PriceMonitoringBounds[0].MaxValidPrice)
	assert.NotEqual(t, md.PriceMonitoringBounds[0].Trigger.Horizon, md2.PriceMonitoringBounds[0].Trigger.Horizon)
	assert.NotEqual(t, md.PriceMonitoringBounds[0].Trigger.Probability, md2.PriceMonitoringBounds[0].Trigger.Probability)
	assert.NotEqual(t, md.PriceMonitoringBounds[0].Trigger.AuctionExtension, md2.PriceMonitoringBounds[0].Trigger.AuctionExtension)
	assert.NotEqual(t, md.PriceMonitoringBounds[0].ReferencePrice, md2.PriceMonitoringBounds[0].ReferencePrice)
	assert.NotEqual(t, md.MarketValueProxy, md2.MarketValueProxy)
	assert.NotEqual(t, md.LiquidityProviderFeeShare[0].Party, md2.LiquidityProviderFeeShare[0].Party)
	assert.NotEqual(t, md.LiquidityProviderFeeShare[0].EquityLikeShare, md2.LiquidityProviderFeeShare[0].EquityLikeShare)
	assert.NotEqual(t, md.LiquidityProviderFeeShare[0].AverageEntryValuation, md2.LiquidityProviderFeeShare[0].AverageEntryValuation)
}
