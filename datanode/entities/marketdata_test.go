// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	types "code.vegaprotocol.io/vega/protos/vega"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestMarketDataFromProto(t *testing.T) {
	t.Run("should parse all valid prices", testParseAllValidPrices)
	t.Run("should return zero for prices if string is empty", testParseEmptyPrices)
	t.Run("should return error if an invalid price string is provided", testParseInvalidPriceString)
	t.Run("should parse valid market data records successfully", testParseMarketDataSuccessfully)
}

func testParseAllValidPrices(t *testing.T) {
	marketdata := types.MarketData{
		MarkPrice:            "1",
		BestBidPrice:         "1",
		BestOfferPrice:       "1",
		BestStaticBidPrice:   "1",
		BestStaticOfferPrice: "1",
		MidPrice:             "1",
		StaticMidPrice:       "1",
		IndicativePrice:      "1",
		TargetStake:          "1",
		SuppliedStake:        "1",
	}

	md, err := entities.MarketDataFromProto(&marketdata, generateTxHash())
	assert.NoError(t, err)
	assert.NotNil(t, md.MarkPrice)
	assert.NotNil(t, md.BestBidPrice)
	assert.NotNil(t, md.BestOfferPrice)
	assert.NotNil(t, md.BestStaticBidPrice)
	assert.NotNil(t, md.BestStaticOfferVolume)
	assert.NotNil(t, md.MidPrice)
	assert.NotNil(t, md.StaticMidPrice)
	assert.NotNil(t, md.IndicativePrice)
	assert.NotNil(t, md.TargetStake)
	assert.NotNil(t, md.SuppliedStake)

	want := decimal.NewFromInt(1)
	assert.True(t, want.Equal(md.MarkPrice))
	assert.True(t, want.Equal(md.BestBidPrice))
	assert.True(t, want.Equal(md.BestOfferPrice))
	assert.True(t, want.Equal(md.BestStaticBidPrice))
	assert.True(t, want.Equal(md.BestStaticOfferPrice))
	assert.True(t, want.Equal(md.MidPrice))
	assert.True(t, want.Equal(md.StaticMidPrice))
	assert.True(t, want.Equal(md.IndicativePrice))
	assert.True(t, want.Equal(md.TargetStake))
	assert.True(t, want.Equal(md.SuppliedStake))
}

func testParseEmptyPrices(t *testing.T) {
	marketdata := types.MarketData{}
	md, err := entities.MarketDataFromProto(&marketdata, generateTxHash())
	assert.NoError(t, err)
	assert.True(t, decimal.Zero.Equals(md.MarkPrice))
	assert.True(t, decimal.Zero.Equals(md.BestBidPrice))
	assert.True(t, decimal.Zero.Equals(md.BestOfferPrice))
	assert.True(t, decimal.Zero.Equals(md.BestStaticBidPrice))
	assert.True(t, decimal.Zero.Equals(md.BestStaticOfferPrice))
	assert.True(t, decimal.Zero.Equals(md.MidPrice))
	assert.True(t, decimal.Zero.Equals(md.StaticMidPrice))
	assert.True(t, decimal.Zero.Equals(md.IndicativePrice))
	assert.True(t, decimal.Zero.Equals(md.TargetStake))
	assert.True(t, decimal.Zero.Equals(md.SuppliedStake))
}

func testParseInvalidPriceString(t *testing.T) {
	type args struct {
		marketdata types.MarketData
	}
	testCases := []struct {
		name string
		args args
	}{
		{
			name: "Invalid Mark Price",
			args: args{
				marketdata: types.MarketData{
					MarkPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Best Bid Price",
			args: args{
				marketdata: types.MarketData{
					BestBidPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Best Offer Price",
			args: args{
				marketdata: types.MarketData{
					BestOfferPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Best Static Bid Price",
			args: args{
				marketdata: types.MarketData{
					BestStaticBidPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Best Static Offer Price",
			args: args{
				marketdata: types.MarketData{
					BestStaticOfferPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Mid Price",
			args: args{
				marketdata: types.MarketData{
					MidPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Static Mid Price",
			args: args{
				marketdata: types.MarketData{
					StaticMidPrice: "Test",
				},
			},
		},
		{
			name: "Invalid Indicative Price",
			args: args{
				marketdata: types.MarketData{
					IndicativePrice: "Test",
				},
			},
		},
		{
			name: "Invalid Target Stake",
			args: args{
				marketdata: types.MarketData{
					TargetStake: "Test",
				},
			},
		},
		{
			name: "Invalid Supplied Stake",
			args: args{
				marketdata: types.MarketData{
					SuppliedStake: "Test",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			md, err := entities.MarketDataFromProto(&tc.args.marketdata, generateTxHash())
			assert.Error(tt, err)
			assert.Nil(tt, md)
		})
	}
}

func testParseMarketDataSuccessfully(t *testing.T) {
	type args struct {
		marketdata types.MarketData
	}

	nextMTM := time.Now()
	zeroTime := time.Unix(0, 0)

	testCases := []struct {
		name string
		args args
		want *entities.MarketData
	}{
		{
			name: "Empty market data",
			args: args{
				marketdata: types.MarketData{},
			},
			want: &entities.MarketData{
				AuctionTrigger:    "AUCTION_TRIGGER_UNSPECIFIED",
				MarketState:       "STATE_UNSPECIFIED",
				MarketTradingMode: "TRADING_MODE_UNSPECIFIED",
				ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
				TxHash:            generateTxHash(),
				NextMarkToMarket:  zeroTime,
			},
		},
		{
			name: "Market data with auction trigger specified",
			args: args{
				marketdata: types.MarketData{
					Trigger:          types.AuctionTrigger_AUCTION_TRIGGER_PRICE,
					NextMarkToMarket: nextMTM.UnixNano(),
				},
			},
			want: &entities.MarketData{
				AuctionTrigger:    "AUCTION_TRIGGER_PRICE",
				MarketState:       "STATE_UNSPECIFIED",
				MarketTradingMode: "TRADING_MODE_UNSPECIFIED",
				ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
				NextMarkToMarket:  nextMTM,
				TxHash:            generateTxHash(),
			},
		},
		{
			name: "Market data with auction trigger and market trading mode specified",
			args: args{
				marketdata: types.MarketData{
					Trigger:           types.AuctionTrigger_AUCTION_TRIGGER_PRICE,
					MarketTradingMode: types.Market_TRADING_MODE_CONTINUOUS,
					NextMarkToMarket:  nextMTM.UnixNano(),
				},
			},
			want: &entities.MarketData{
				AuctionTrigger:    "AUCTION_TRIGGER_PRICE",
				MarketTradingMode: "TRADING_MODE_CONTINUOUS",
				MarketState:       "STATE_UNSPECIFIED",
				ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
				NextMarkToMarket:  nextMTM,
				TxHash:            generateTxHash(),
			},
		},
		{
			name: "Market data with best bid and best offer specified",
			args: args{
				marketdata: types.MarketData{
					BestBidPrice:      "100.0",
					BestOfferPrice:    "110.0",
					Trigger:           types.AuctionTrigger_AUCTION_TRIGGER_PRICE,
					MarketTradingMode: types.Market_TRADING_MODE_CONTINUOUS,
					MarketState:       types.Market_STATE_ACTIVE,
					NextMarkToMarket:  nextMTM.UnixNano(),
				},
			},
			want: &entities.MarketData{
				BestBidPrice:      decimal.NewFromFloat(100.0),
				BestOfferPrice:    decimal.NewFromFloat(110.0),
				AuctionTrigger:    "AUCTION_TRIGGER_PRICE",
				MarketState:       "STATE_ACTIVE",
				MarketTradingMode: "TRADING_MODE_CONTINUOUS",
				ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
				TxHash:            generateTxHash(),
				NextMarkToMarket:  nextMTM,
			},
		},
		{
			name: "Market data with best bid and best offer specified and price monitoring bounds",
			args: args{
				marketdata: types.MarketData{
					BestBidPrice:      "100.0",
					BestOfferPrice:    "110.0",
					Trigger:           types.AuctionTrigger_AUCTION_TRIGGER_PRICE,
					MarketTradingMode: types.Market_TRADING_MODE_CONTINUOUS,
					PriceMonitoringBounds: []*types.PriceMonitoringBounds{
						{
							MinValidPrice: "100",
							MaxValidPrice: "200",
						},
					},
					NextMarkToMarket: nextMTM.UnixNano(),
				},
			},
			want: &entities.MarketData{
				BestBidPrice:      decimal.NewFromFloat(100.0),
				BestOfferPrice:    decimal.NewFromFloat(110.0),
				AuctionTrigger:    "AUCTION_TRIGGER_PRICE",
				MarketTradingMode: "TRADING_MODE_CONTINUOUS",
				MarketState:       "STATE_UNSPECIFIED",
				ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
				PriceMonitoringBounds: []*entities.PriceMonitoringBound{
					{
						MinValidPrice:  num.NewUint(100),
						MaxValidPrice:  num.NewUint(200),
						ReferencePrice: num.NewUint(0),
					},
				},
				TxHash:           generateTxHash(),
				NextMarkToMarket: nextMTM,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			got, err := entities.MarketDataFromProto(&tc.args.marketdata, tc.want.TxHash)
			assert.NoError(tt, err)
			assert.True(tt, tc.want.Equal(*got))
		})
	}
}

func TestPriceMonitoringBound_MarshalJSON(t *testing.T) {
	min, _ := num.UintFromString("1", 10)
	max, _ := num.UintFromString("2", 10)
	ref, _ := num.UintFromString("3", 10)

	bounds := &entities.PriceMonitoringBound{
		MinValidPrice: min,
		MaxValidPrice: max,
		Trigger: entities.PriceMonitoringTrigger{
			Horizon:          100,
			Probability:      decimal.NewFromFloat(0.5),
			AuctionExtension: 200,
		},
		ReferencePrice: ref,
	}

	bs, err := json.Marshal(bounds)
	require.NoError(t, err)

	want := []byte(`{"minValidPrice":"1","maxValidPrice":"2","trigger":{"horizon":100,"probability":"0.5","auctionExtension":200},"referencePrice":"3"}`)
	assert.Equal(t, want, bs)
}

func TestPriceMonitoringBound_UnmarshalJSON(t *testing.T) {
	bs := []byte(`{"minValidPrice":"1","maxValidPrice":"2","trigger":{"horizon":100,"probability":"0.5","auctionExtension":200},"referencePrice":"3"}`)
	var bounds entities.PriceMonitoringBound
	err := json.Unmarshal(bs, &bounds)
	require.NoError(t, err)

	min, _ := num.UintFromString("1", 10)
	max, _ := num.UintFromString("2", 10)
	ref, _ := num.UintFromString("3", 10)

	want := entities.PriceMonitoringBound{
		MinValidPrice: min,
		MaxValidPrice: max,
		Trigger: entities.PriceMonitoringTrigger{
			Horizon:          100,
			Probability:      decimal.NewFromFloat(0.5),
			AuctionExtension: 200,
		},
		ReferencePrice: ref,
	}
	assert.Equal(t, want, bounds)
}
