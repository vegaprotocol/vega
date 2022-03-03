package entities

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

var ErrMarketDataIntegerOverflow = errors.New("integer overflow encountered when converting market data for persistence")

// MarketData represents a market data record that is stored in the SQL database
type MarketData struct {
	// Mark price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	MarkPrice decimal.Decimal
	// Highest price level on an order book for buy orders, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	BestBidPrice decimal.Decimal
	// Aggregated volume being bid at the best bid price
	BestBidVolume int64
	// Aggregated volume being bid at the best bid price
	BestOfferPrice decimal.Decimal
	// Aggregated volume being offered at the best offer price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	BestOfferVolume int64
	// Highest price on the order book for buy orders not including pegged orders
	BestStaticBidPrice decimal.Decimal
	// Total volume at the best static bid price excluding pegged orders
	BestStaticBidVolume int64
	// Lowest price on the order book for sell orders not including pegged orders
	BestStaticOfferPrice decimal.Decimal
	// Total volume at the best static offer price excluding pegged orders
	BestStaticOfferVolume int64
	// Arithmetic average of the best bid price and best offer price, as an integer, for example `123456` is a correctly
	// formatted price of `1.23456` assuming market configured to 5 decimal places
	MidPrice decimal.Decimal
	// Arithmetic average of the best static bid price and best static offer price
	StaticMidPrice decimal.Decimal
	// Market identifier for the data
	Market []byte
	// The sum of the size of all positions greater than 0 on the market
	OpenInterest int64
	// Time in seconds until the end of the auction (0 if currently not in auction period)
	AuctionEnd int64
	// Time until next auction (used in FBA's) - currently always 0
	AuctionStart int64
	// Indicative price (zero if not in auction)
	IndicativePrice decimal.Decimal
	// Indicative volume (zero if not in auction)
	IndicativeVolume int64
	// The current trading mode for the market
	MarketTradingMode string
	// When a market is in an auction trading mode, this field indicates what triggered the auction
	AuctionTrigger string
	// When a market auction is extended, this field indicates what caused the extension
	ExtensionTrigger string
	// Targeted stake for the given market
	TargetStake decimal.Decimal
	// Available stake for the given market
	SuppliedStake decimal.Decimal
	// One or more price monitoring bounds for the current timestamp
	PriceMonitoringBounds []*PriceMonitoringBound
	// the market value proxy
	MarketValueProxy string
	// the equity like share of liquidity fee for each liquidity provider
	LiquidityProviderFeeShares []*LiquidityProviderFeeShare
	// Vega Block time at which the data was received from Vega Node
	VegaTime time.Time
	// SeqNum is the order in which the market data was received in the block
	SeqNum uint64
}

type PriceMonitoringTrigger struct {
	Horizon          uint64          `json:"horizon"`
	Probability      decimal.Decimal `json:"probability"`
	AuctionExtension uint64          `json:"auctionExtension"`
}

func (trigger PriceMonitoringTrigger) Equals(other PriceMonitoringTrigger) bool {
	return trigger.Horizon == other.Horizon &&
		trigger.Probability == other.Probability &&
		trigger.AuctionExtension == other.AuctionExtension
}

type PriceMonitoringBound struct {
	MinValidPrice  uint64                 `json:"minValidPrice"`
	MaxValidPrice  uint64                 `json:"maxValidPrice"`
	Trigger        PriceMonitoringTrigger `json:"trigger"`
	ReferencePrice uint64                 `json:"referencePrice"`
}

func (bound PriceMonitoringBound) Equals(other PriceMonitoringBound) bool {
	return bound.MinValidPrice == other.MinValidPrice &&
		bound.MaxValidPrice == other.MaxValidPrice &&
		bound.Trigger.Equals(other.Trigger)
}

type LiquidityProviderFeeShare struct {
	Party                 string `json:"party"`
	EquityLikeShare       uint64 `json:"equityLikeShare"`
	AverageEntryValuation uint64 `json:"averageEntryValuation"`
}

func (fee LiquidityProviderFeeShare) Equals(other LiquidityProviderFeeShare) bool {
	return fee.Party == other.Party &&
		fee.EquityLikeShare == other.EquityLikeShare &&
		fee.AverageEntryValuation == other.AverageEntryValuation
}

func MarketDataFromProto(data *types.MarketData) (*MarketData, error) {
	var mark, bid, offer, staticBid, staticOffer, mid, staticMid, indicative, targetStake, suppliedStake decimal.Decimal
	var err error
	var marketID []byte

	if marketID, err = hex.DecodeString(data.Market); err != nil {
		return nil, nil
	}
	if mark, err = parseDecimal(data.MarkPrice); err != nil {
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

	if data.BestBidVolume > math.MaxInt64 || data.BestOfferVolume > math.MaxInt64 || data.BestStaticBidVolume > math.MaxInt64 ||
		data.BestStaticOfferVolume > math.MaxInt64 || data.OpenInterest > math.MaxInt64 || data.IndicativeVolume > math.MaxInt64 {
		return nil, ErrMarketDataIntegerOverflow
	}

	bestBidVolume := int64(data.BestBidVolume)
	bestOfferVolume := int64(data.BestOfferVolume)
	bestStaticBidVolume := int64(data.BestStaticBidVolume)
	bestStaticOfferVolume := int64(data.BestStaticOfferVolume)
	openInterest := int64(data.OpenInterest)
	indicativeVolume := int64(data.IndicativeVolume)

	marketData := &MarketData{
		MarkPrice:                  mark,
		BestBidPrice:               bid,
		BestBidVolume:              bestBidVolume,
		BestOfferPrice:             offer,
		BestOfferVolume:            bestOfferVolume,
		BestStaticBidPrice:         staticBid,
		BestStaticBidVolume:        bestStaticBidVolume,
		BestStaticOfferPrice:       staticOffer,
		BestStaticOfferVolume:      bestStaticOfferVolume,
		MidPrice:                   mid,
		StaticMidPrice:             staticMid,
		Market:                     marketID,
		OpenInterest:               openInterest,
		AuctionEnd:                 data.AuctionEnd,
		AuctionStart:               data.AuctionStart,
		IndicativePrice:            indicative,
		IndicativeVolume:           indicativeVolume,
		MarketTradingMode:          data.MarketTradingMode.String(),
		AuctionTrigger:             data.Trigger.String(),
		ExtensionTrigger:           data.ExtensionTrigger.String(),
		TargetStake:                targetStake,
		SuppliedStake:              suppliedStake,
		PriceMonitoringBounds:      parsePriceMonitoringBounds(data.PriceMonitoringBounds),
		MarketValueProxy:           data.MarketValueProxy,
		LiquidityProviderFeeShares: parseLiquidityProviderFeeShares(data.LiquidityProviderFeeShare),
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

func parsePriceMonitoringBounds(bounds []*types.PriceMonitoringBounds) []*PriceMonitoringBound {
	if len(bounds) == 0 {
		return nil
	}

	results := make([]*PriceMonitoringBound, 0, len(bounds))

	for _, b := range bounds {
		results = append(results, priceMonitoringBoundsFromProto(b))
	}

	return results
}

func parseLiquidityProviderFeeShares(shares []*types.LiquidityProviderFeeShare) []*LiquidityProviderFeeShare {
	if len(shares) == 0 {
		return nil
	}

	results := make([]*LiquidityProviderFeeShare, 0, len(shares))

	for _, s := range shares {
		results = append(results, liquidityProviderFeeShareFromProto(s))
	}

	return results
}

func priceMonitoringBoundsFromProto(bounds *types.PriceMonitoringBounds) *PriceMonitoringBound {
	if bounds == nil {
		return nil
	}

	minValidPrice, _ := strconv.ParseUint(bounds.MinValidPrice, 10, 64)
	maxValidPrice, _ := strconv.ParseUint(bounds.MaxValidPrice, 10, 64)

	referencePrice, _ := strconv.ParseUint(bounds.ReferencePrice, 10, 64)

	return &PriceMonitoringBound{
		MinValidPrice:  minValidPrice,
		MaxValidPrice:  maxValidPrice,
		Trigger:        priceMonitoringTriggerFromProto(bounds.Trigger),
		ReferencePrice: referencePrice,
	}
}

func priceMonitoringTriggerFromProto(trigger *types.PriceMonitoringTrigger) PriceMonitoringTrigger {
	if trigger == nil {
		return PriceMonitoringTrigger{}
	}

	probability, _ := decimal.NewFromString(trigger.Probability)

	return PriceMonitoringTrigger{
		Horizon:          uint64(trigger.Horizon),
		Probability:      probability,
		AuctionExtension: uint64(trigger.AuctionExtension),
	}
}

func liquidityProviderFeeShareFromProto(feeShare *types.LiquidityProviderFeeShare) *LiquidityProviderFeeShare {
	if feeShare == nil {
		return nil
	}

	equityLikeShare, _ := strconv.ParseUint(feeShare.EquityLikeShare, 10, 64)
	averageEntryValuation, _ := strconv.ParseUint(feeShare.AverageEntryValuation, 10, 64)

	return &LiquidityProviderFeeShare{
		Party:                 feeShare.Party,
		EquityLikeShare:       equityLikeShare,
		AverageEntryValuation: averageEntryValuation,
	}
}

func (md MarketData) Equal(other MarketData) bool {
	return md.MarkPrice.Equals(other.MarkPrice) &&
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
		bytes.Equal(md.Market, other.Market) &&
		md.MarketTradingMode == other.MarketTradingMode &&
		md.AuctionTrigger == other.AuctionTrigger &&
		md.ExtensionTrigger == other.ExtensionTrigger &&
		md.MarketValueProxy == other.MarketValueProxy &&
		priceMonitoringBoundsMatches(md.PriceMonitoringBounds, other.PriceMonitoringBounds) &&
		liquidityProviderFeeShareMatches(md.LiquidityProviderFeeShares, other.LiquidityProviderFeeShares)
}

func priceMonitoringBoundsMatches(bounds, other []*PriceMonitoringBound) bool {
	if len(bounds) != len(other) {
		return false
	}

	for i, bound := range bounds {
		if !bound.Equals(*other[i]) {
			return false
		}
	}

	return true
}

func liquidityProviderFeeShareMatches(feeShares, other []*LiquidityProviderFeeShare) bool {
	if len(feeShares) != len(other) {
		return false
	}

	for i, fee := range feeShares {
		if !fee.Equals(*other[i]) {
			return false
		}
	}

	return true
}

func (md MarketData) ToProto() *types.MarketData {
	result := types.MarketData{
		MarkPrice:                 md.MarkPrice.String(),
		BestBidPrice:              md.BestBidPrice.String(),
		BestBidVolume:             uint64(md.BestBidVolume),
		BestOfferPrice:            md.BestOfferPrice.String(),
		BestOfferVolume:           uint64(md.BestOfferVolume),
		BestStaticBidPrice:        md.BestStaticBidPrice.String(),
		BestStaticBidVolume:       uint64(md.BestStaticBidVolume),
		BestStaticOfferPrice:      md.BestStaticOfferPrice.String(),
		BestStaticOfferVolume:     uint64(md.BestStaticOfferVolume),
		MidPrice:                  md.MidPrice.String(),
		StaticMidPrice:            md.StaticMidPrice.String(),
		Market:                    hex.EncodeToString(md.Market),
		Timestamp:                 md.VegaTime.UnixNano(),
		OpenInterest:              uint64(md.OpenInterest),
		AuctionEnd:                md.AuctionEnd,
		AuctionStart:              md.AuctionStart,
		IndicativePrice:           md.IndicativePrice.String(),
		IndicativeVolume:          uint64(md.IndicativeVolume),
		MarketTradingMode:         types.Market_TradingMode(types.Market_TradingMode_value[md.MarketTradingMode]),
		Trigger:                   types.AuctionTrigger(types.Market_TradingMode_value[md.AuctionTrigger]),
		ExtensionTrigger:          types.AuctionTrigger(types.Market_TradingMode_value[md.ExtensionTrigger]),
		TargetStake:               md.TargetStake.String(),
		SuppliedStake:             md.SuppliedStake.String(),
		PriceMonitoringBounds:     priceMonitoringBoundsToProto(md.PriceMonitoringBounds),
		MarketValueProxy:          md.MarketValueProxy,
		LiquidityProviderFeeShare: liquidityProviderFeeSharesToProto(md.LiquidityProviderFeeShares),
	}

	return &result
}

func priceMonitoringBoundsToProto(bounds []*PriceMonitoringBound) []*types.PriceMonitoringBounds {
	if len(bounds) == 0 {
		return nil
	}

	results := make([]*types.PriceMonitoringBounds, 0, len(bounds))

	for _, bound := range bounds {
		protoBound := types.PriceMonitoringBounds{
			MinValidPrice:  fmt.Sprintf("%d", bound.MinValidPrice),
			MaxValidPrice:  fmt.Sprintf("%d", bound.MaxValidPrice),
			Trigger:        priceMonitoringTriggerToProto(bound.Trigger),
			ReferencePrice: fmt.Sprintf("%d", bound.ReferencePrice),
		}

		results = append(results, &protoBound)
	}

	return results
}

func liquidityProviderFeeSharesToProto(feeShares []*LiquidityProviderFeeShare) []*types.LiquidityProviderFeeShare {
	if len(feeShares) == 0 {
		return nil
	}

	results := make([]*types.LiquidityProviderFeeShare, 0, len(feeShares))

	for _, feeShare := range feeShares {
		protoFeeShare := types.LiquidityProviderFeeShare{
			Party:                 feeShare.Party,
			EquityLikeShare:       fmt.Sprintf("%d", feeShare.EquityLikeShare),
			AverageEntryValuation: fmt.Sprintf("%d", feeShare.AverageEntryValuation),
		}

		results = append(results, &protoFeeShare)
	}

	return results
}

func priceMonitoringTriggerToProto(trigger PriceMonitoringTrigger) *types.PriceMonitoringTrigger {
	return &types.PriceMonitoringTrigger{
		Horizon:          int64(trigger.Horizon),
		Probability:      trigger.Probability.String(),
		AuctionExtension: int64(trigger.AuctionExtension),
	}
}
