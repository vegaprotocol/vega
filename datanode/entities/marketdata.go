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

package entities

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

var ErrMarketDataIntegerOverflow = errors.New("integer overflow encountered when converting market data for persistence")

// MarketData represents a market data record that is stored in the SQL database.
type MarketData struct {
	// Mark price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	MarkPrice decimal.Decimal
	// Highest price level on an order book for buy orders, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	BestBidPrice decimal.Decimal
	// Aggregated volume being bid at the best bid price
	BestBidVolume uint64
	// Aggregated volume being bid at the best bid price
	BestOfferPrice decimal.Decimal
	// Aggregated volume being offered at the best offer price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	BestOfferVolume uint64
	// Highest price on the order book for buy orders not including pegged orders
	BestStaticBidPrice decimal.Decimal
	// Total volume at the best static bid price excluding pegged orders
	BestStaticBidVolume uint64
	// Lowest price on the order book for sell orders not including pegged orders
	BestStaticOfferPrice decimal.Decimal
	// Total volume at the best static offer price excluding pegged orders
	BestStaticOfferVolume uint64
	// Arithmetic average of the best bid price and best offer price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	MidPrice decimal.Decimal
	// Arithmetic average of the best static bid price and best static offer price
	StaticMidPrice decimal.Decimal
	// Market identifier for the data
	Market MarketID
	// The sum of the size of all positions greater than 0 on the market
	OpenInterest uint64
	// Time in seconds until the end of the auction (0 if currently not in auction period)
	AuctionEnd int64
	// Time until next auction (used in FBA's) - currently always 0
	AuctionStart int64
	// Indicative price (zero if not in auction)
	IndicativePrice decimal.Decimal
	// Indicative volume (zero if not in auction)
	IndicativeVolume uint64
	// The current trading mode for the market
	MarketTradingMode string
	// The current trading mode for the market
	MarketState string
	// When a market is in an auction trading mode, this field indicates what triggered the auction
	AuctionTrigger string
	// When a market auction is extended, this field indicates what caused the extension
	ExtensionTrigger string
	// Targeted stake for the given market
	TargetStake num.Decimal
	// Available stake for the given market
	SuppliedStake num.Decimal
	// One or more price monitoring bounds for the current timestamp
	PriceMonitoringBounds []*types.PriceMonitoringBounds
	// the market value proxy
	MarketValueProxy string
	// the equity like share of liquidity fee for each liquidity provider
	LiquidityProviderFeeShares []*types.LiquidityProviderFeeShare
	// the SLA statistics for each liquidity provider
	LiquidityProviderSLA []*types.LiquidityProviderSLA
	// A synthetic time created which is the sum of vega_time + (seq num * Microsecond)
	SyntheticTime time.Time
	// Transaction which caused this update
	TxHash TxHash
	// Vega Block time at which the data was received from Vega Node
	VegaTime time.Time
	// SeqNum is the order in which the market data was received in the block
	SeqNum uint64
	// NextMarkToMarket is the next timestamp when the market will be marked to market
	NextMarkToMarket time.Time
	// market growth for the last market window
	MarketGrowth num.Decimal
	// Last traded price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	LastTradedPrice num.Decimal
	// Data specific to the product type
	ProductData *ProductData
	// NextNetworkCloseout is the time at which the network will attempt its next closeout order.
	NextNetworkCloseout time.Time
	// The methodology used for the calculation of the mark price.
	MarkPriceType string
	// The internal state of the mark price composite price.
	MarkPriceState *CompositePriceState
	// The state of the active protocol
	ActiveProtocolAutomatedPurchase *vega.ProtocolAutomatedPurchaseData
}

type PriceMonitoringTrigger struct {
	Horizon          uint64          `json:"horizon"`
	Probability      decimal.Decimal `json:"probability"`
	AuctionExtension uint64          `json:"auctionExtension"`
}

func (trigger PriceMonitoringTrigger) Equals(other PriceMonitoringTrigger) bool {
	return trigger.Horizon == other.Horizon &&
		trigger.Probability.Equal(other.Probability) &&
		trigger.AuctionExtension == other.AuctionExtension
}

func (trigger PriceMonitoringTrigger) ToProto() *types.PriceMonitoringTrigger {
	return &types.PriceMonitoringTrigger{
		Horizon:          int64(trigger.Horizon),
		Probability:      trigger.Probability.String(),
		AuctionExtension: int64(trigger.AuctionExtension),
	}
}

func MarketDataFromProto(data *types.MarketData, txHash TxHash) (*MarketData, error) {
	var mark, bid, offer, staticBid, staticOffer, mid, staticMid, indicative, targetStake, suppliedStake, growth, lastTradedPrice num.Decimal
	var err error

	if mark, err = parseDecimal(data.MarkPrice); err != nil {
		return nil, err
	}
	if lastTradedPrice, err = parseDecimal(data.LastTradedPrice); err != nil {
		return nil, err
	}
	if bid, err = parseDecimal(data.BestBidPrice); err != nil {
		return nil, err
	}
	if offer, err = parseDecimal(data.BestOfferPrice); err != nil {
		return nil, err
	}
	if staticBid, err = parseDecimal(data.BestStaticBidPrice); err != nil {
		return nil, err
	}
	if staticOffer, err = parseDecimal(data.BestStaticOfferPrice); err != nil {
		return nil, err
	}
	if mid, err = parseDecimal(data.MidPrice); err != nil {
		return nil, err
	}
	if staticMid, err = parseDecimal(data.StaticMidPrice); err != nil {
		return nil, err
	}
	if indicative, err = parseDecimal(data.IndicativePrice); err != nil {
		return nil, err
	}
	if targetStake, err = parseDecimal(data.TargetStake); err != nil {
		return nil, err
	}
	if suppliedStake, err = parseDecimal(data.SuppliedStake); err != nil {
		return nil, err
	}
	if growth, err = parseDecimal(data.MarketGrowth); err != nil {
		return nil, err
	}
	nextMTM := time.Unix(0, data.NextMarkToMarket)
	nextNC := time.Unix(0, data.NextNetworkCloseout)

	marketData := &MarketData{
		LastTradedPrice:                 lastTradedPrice,
		MarkPrice:                       mark,
		BestBidPrice:                    bid,
		BestBidVolume:                   data.BestBidVolume,
		BestOfferPrice:                  offer,
		BestOfferVolume:                 data.BestOfferVolume,
		BestStaticBidPrice:              staticBid,
		BestStaticBidVolume:             data.BestStaticBidVolume,
		BestStaticOfferPrice:            staticOffer,
		BestStaticOfferVolume:           data.BestStaticOfferVolume,
		MidPrice:                        mid,
		StaticMidPrice:                  staticMid,
		Market:                          MarketID(data.Market),
		OpenInterest:                    data.OpenInterest,
		AuctionEnd:                      data.AuctionEnd,
		AuctionStart:                    data.AuctionStart,
		IndicativePrice:                 indicative,
		IndicativeVolume:                data.IndicativeVolume,
		MarketState:                     data.MarketState.String(),
		MarketTradingMode:               data.MarketTradingMode.String(),
		AuctionTrigger:                  data.Trigger.String(),
		ExtensionTrigger:                data.ExtensionTrigger.String(),
		TargetStake:                     targetStake,
		SuppliedStake:                   suppliedStake,
		PriceMonitoringBounds:           data.PriceMonitoringBounds,
		MarketValueProxy:                data.MarketValueProxy,
		LiquidityProviderFeeShares:      data.LiquidityProviderFeeShare,
		LiquidityProviderSLA:            data.LiquidityProviderSla,
		TxHash:                          txHash,
		NextMarkToMarket:                nextMTM,
		MarketGrowth:                    growth,
		NextNetworkCloseout:             nextNC,
		MarkPriceType:                   data.MarkPriceType.String(),
		ActiveProtocolAutomatedPurchase: data.ActiveProtocolAutomatedPurchase,
	}

	if data.MarkPriceState != nil {
		marketData.MarkPriceState = &CompositePriceState{data.MarkPriceState}
	}

	if data.ProductData != nil {
		marketData.ProductData = &ProductData{data.ProductData}
	}

	return marketData, nil
}

func parseDecimal(input string) (decimal.Decimal, error) {
	if input == "" {
		return decimal.Zero, nil
	}

	v, err := decimal.NewFromString(input)
	if err != nil {
		return decimal.Zero, err
	}

	return v, nil
}

func (md MarketData) Equal(other MarketData) bool {
	productData1 := []byte{}
	productData2 := []byte{}
	if md.ProductData != nil {
		productData1, _ = md.ProductData.MarshalJSON()
	}
	if other.ProductData != nil {
		productData2, _ = other.ProductData.MarshalJSON()
	}

	markPriceState1 := []byte{}
	markPriceState2 := []byte{}
	if md.MarkPriceState != nil {
		markPriceState1, _ = md.MarkPriceState.MarshalJSON()
	}
	if other.MarkPriceState != nil {
		markPriceState2, _ = other.MarkPriceState.MarshalJSON()
	}

	papStateIsEqual := (md.ActiveProtocolAutomatedPurchase == nil) == (other.ActiveProtocolAutomatedPurchase == nil) &&
		(md.ActiveProtocolAutomatedPurchase == nil || md.ActiveProtocolAutomatedPurchase.Id == other.ActiveProtocolAutomatedPurchase.Id && md.ActiveProtocolAutomatedPurchase.OrderId == other.ActiveProtocolAutomatedPurchase.OrderId)

	return md.LastTradedPrice.Equals(other.LastTradedPrice) &&
		md.MarkPrice.Equals(other.MarkPrice) &&
		md.BestBidPrice.Equals(other.BestBidPrice) &&
		md.BestOfferPrice.Equals(other.BestOfferPrice) &&
		md.BestStaticBidPrice.Equals(other.BestStaticBidPrice) &&
		md.BestStaticOfferPrice.Equals(other.BestStaticOfferPrice) &&
		md.MidPrice.Equals(other.MidPrice) &&
		md.StaticMidPrice.Equals(other.StaticMidPrice) &&
		md.IndicativePrice.Equals(other.IndicativePrice) &&
		md.TargetStake.Equals(other.TargetStake) &&
		md.SuppliedStake.Equals(other.SuppliedStake) &&
		md.BestBidVolume == other.BestBidVolume &&
		md.BestOfferVolume == other.BestOfferVolume &&
		md.BestStaticBidVolume == other.BestStaticBidVolume &&
		md.BestStaticOfferVolume == other.BestStaticOfferVolume &&
		md.OpenInterest == other.OpenInterest &&
		md.AuctionEnd == other.AuctionEnd &&
		md.AuctionStart == other.AuctionStart &&
		md.IndicativeVolume == other.IndicativeVolume &&
		md.Market == other.Market &&
		md.MarketTradingMode == other.MarketTradingMode &&
		md.AuctionTrigger == other.AuctionTrigger &&
		md.ExtensionTrigger == other.ExtensionTrigger &&
		md.MarketValueProxy == other.MarketValueProxy &&
		priceMonitoringBoundsMatches(md.PriceMonitoringBounds, other.PriceMonitoringBounds) &&
		liquidityProviderFeeShareMatches(md.LiquidityProviderFeeShares, other.LiquidityProviderFeeShares) &&
		liquidityProviderSLAMatches(md.LiquidityProviderSLA, other.LiquidityProviderSLA) &&
		md.TxHash == other.TxHash &&
		md.MarketState == other.MarketState &&
		md.NextMarkToMarket.Equal(other.NextMarkToMarket) &&
		md.MarketGrowth.Equal(other.MarketGrowth) &&
		bytes.Equal(productData1, productData2) &&
		md.NextNetworkCloseout.Equal(other.NextNetworkCloseout) &&
		md.MarkPriceType == other.MarkPriceType &&
		bytes.Equal(markPriceState1, markPriceState2) && papStateIsEqual
}

func priceMonitoringBoundsMatches(bounds, other []*types.PriceMonitoringBounds) bool {
	if len(bounds) != len(other) {
		return false
	}

	for i, bound := range bounds {
		if bound.Trigger == nil && other[i].Trigger != nil || bound.Trigger != nil && other[i].Trigger == nil {
			return false
		}

		if bound.MinValidPrice != other[i].MinValidPrice ||
			bound.MaxValidPrice != other[i].MaxValidPrice ||
			bound.ReferencePrice != other[i].ReferencePrice || bound.Active != other[i].Active {
			return false
		}

		if bound.Trigger != nil && other[i].Trigger != nil {
			if bound.Trigger.Probability != other[i].Trigger.Probability ||
				bound.Trigger.Horizon != other[i].Trigger.Horizon ||
				bound.Trigger.AuctionExtension != other[i].Trigger.AuctionExtension {
				return false
			}
		}
	}

	return true
}

func liquidityProviderFeeShareMatches(feeShares, other []*types.LiquidityProviderFeeShare) bool {
	if len(feeShares) != len(other) {
		return false
	}

	for i, fee := range feeShares {
		if fee.EquityLikeShare != other[i].EquityLikeShare ||
			fee.AverageEntryValuation != other[i].AverageEntryValuation ||
			fee.AverageScore != other[i].AverageScore ||
			fee.Party != other[i].Party {
			return false
		}
	}

	return true
}

func liquidityProviderSLAMatches(slas, other []*types.LiquidityProviderSLA) bool {
	if len(slas) != len(other) {
		return false
	}

	for i, sla := range slas {
		if sla.CurrentEpochFractionOfTimeOnBook != other[i].CurrentEpochFractionOfTimeOnBook ||
			sla.LastEpochBondPenalty != other[i].LastEpochBondPenalty ||
			sla.LastEpochFeePenalty != other[i].LastEpochFeePenalty ||
			sla.LastEpochFractionOfTimeOnBook != other[i].LastEpochFractionOfTimeOnBook {
			return false
		}
	}

	return true
}

func (md MarketData) ToProto() *types.MarketData {
	result := types.MarketData{
		LastTradedPrice:                 md.LastTradedPrice.String(),
		MarkPrice:                       md.MarkPrice.String(),
		BestBidPrice:                    md.BestBidPrice.String(),
		BestBidVolume:                   md.BestBidVolume,
		BestOfferPrice:                  md.BestOfferPrice.String(),
		BestOfferVolume:                 md.BestOfferVolume,
		BestStaticBidPrice:              md.BestStaticBidPrice.String(),
		BestStaticBidVolume:             md.BestStaticBidVolume,
		BestStaticOfferPrice:            md.BestStaticOfferPrice.String(),
		BestStaticOfferVolume:           md.BestStaticOfferVolume,
		MidPrice:                        md.MidPrice.String(),
		StaticMidPrice:                  md.StaticMidPrice.String(),
		Market:                          md.Market.String(),
		Timestamp:                       md.VegaTime.UnixNano(),
		OpenInterest:                    md.OpenInterest,
		AuctionEnd:                      md.AuctionEnd,
		AuctionStart:                    md.AuctionStart,
		IndicativePrice:                 md.IndicativePrice.String(),
		IndicativeVolume:                md.IndicativeVolume,
		MarketState:                     types.Market_State(types.Market_State_value[md.MarketState]),
		MarketTradingMode:               types.Market_TradingMode(types.Market_TradingMode_value[md.MarketTradingMode]),
		Trigger:                         types.AuctionTrigger(types.AuctionTrigger_value[md.AuctionTrigger]),
		ExtensionTrigger:                types.AuctionTrigger(types.AuctionTrigger_value[md.ExtensionTrigger]),
		TargetStake:                     md.TargetStake.String(),
		SuppliedStake:                   md.SuppliedStake.String(),
		PriceMonitoringBounds:           md.PriceMonitoringBounds,
		MarketValueProxy:                md.MarketValueProxy,
		LiquidityProviderFeeShare:       md.LiquidityProviderFeeShares,
		LiquidityProviderSla:            md.LiquidityProviderSLA,
		NextMarkToMarket:                md.NextMarkToMarket.UnixNano(),
		MarketGrowth:                    md.MarketGrowth.String(),
		NextNetworkCloseout:             md.NextNetworkCloseout.UnixNano(),
		MarkPriceType:                   types.CompositePriceType(types.CompositePriceType_value[md.MarkPriceType]),
		ActiveProtocolAutomatedPurchase: md.ActiveProtocolAutomatedPurchase,
	}

	if md.ProductData != nil {
		result.ProductData = md.ProductData.ProductData
	}

	if md.MarkPriceState != nil {
		result.MarkPriceState = md.MarkPriceState.CompositePriceState
	}

	return &result
}

func (md MarketData) Cursor() *Cursor {
	return NewCursor(MarketDataCursor{md.SyntheticTime}.String())
}

func (md MarketData) ToProtoEdge(_ ...any) (*v2.MarketDataEdge, error) {
	return &v2.MarketDataEdge{
		Node:   md.ToProto(),
		Cursor: md.Cursor().Encode(),
	}, nil
}

type MarketDataCursor struct {
	SyntheticTime time.Time `json:"synthetic_time"`
}

func (c MarketDataCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal market data cursor: %w", err))
	}
	return string(bs)
}

func (c *MarketDataCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}
