//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import "code.vegaprotocol.io/vega/proto"

type Market = proto.Market
type MarketData = proto.MarketData
type Instrument = proto.Instrument
type Instrument_Future = proto.Instrument_Future
type Future = proto.Future
type Market_Continuous = proto.Market_Continuous
type Market_Discrete = proto.Market_Discrete
type TradableInstrument = proto.TradableInstrument
type TradableInstrument_LogNormalRiskModel = proto.TradableInstrument_LogNormalRiskModel
type TradableInstrument_SimpleRiskModel = proto.TradableInstrument_SimpleRiskModel
type LiquidityProviderFeeShare = proto.LiquidityProviderFeeShare
type AuctionDuration = proto.AuctionDuration
type Fees = proto.Fees
type LogNormalRiskModel = proto.LogNormalRiskModel
type FeeFactors = proto.FeeFactors
type LogNormalModelParams = proto.LogNormalModelParams
type Price = proto.Price
type Timestamp = proto.Timestamp
type InstrumentMetadata = proto.InstrumentMetadata
type OracleSpecToFutureBinding = proto.OracleSpecToFutureBinding
type SimpleRiskModel = proto.SimpleRiskModel
type ContinuousTrading = proto.ContinuousTrading
type SimpleModelParams = proto.SimpleModelParams
type MarketTimestamps = proto.MarketTimestamps

type Market_TradingMode = proto.Market_TradingMode

const (
	// Default value, this is invalid
	Market_TRADING_MODE_UNSPECIFIED Market_TradingMode = 0
	// Normal trading
	Market_TRADING_MODE_CONTINUOUS Market_TradingMode = 1
	// Auction trading (FBA)
	Market_TRADING_MODE_BATCH_AUCTION Market_TradingMode = 2
	// Opening auction
	Market_TRADING_MODE_OPENING_AUCTION Market_TradingMode = 3
	// Auction triggered by monitoring
	Market_TRADING_MODE_MONITORING_AUCTION Market_TradingMode = 4
)

type Market_State = proto.Market_State

const (
	// Default value, invalid
	Market_STATE_UNSPECIFIED Market_State = 0
	// The Governance proposal valid and accepted
	Market_STATE_PROPOSED Market_State = 1
	// Outcome of governance votes is to reject the market
	Market_STATE_REJECTED Market_State = 2
	// Governance vote passes/wins
	Market_STATE_PENDING Market_State = 3
	// Market triggers cancellation condition or governance
	// votes to close before market becomes Active
	Market_STATE_CANCELLED Market_State = 4
	// Enactment date reached and usual auction exit checks pass
	Market_STATE_ACTIVE Market_State = 5
	// Price monitoring or liquidity monitoring trigger
	Market_STATE_SUSPENDED Market_State = 6
	// Governance vote (to close)
	Market_STATE_CLOSED Market_State = 7
	// Defined by the product (i.e. from a product parameter,
	// specified in market definition, giving close date/time)
	Market_STATE_TRADING_TERMINATED Market_State = 8
	// Settlement triggered and completed as defined by product
	Market_STATE_SETTLED Market_State = 9
)

type AuctionTrigger = proto.AuctionTrigger

const (
	// Default value for AuctionTrigger, no auction triggered
	AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED AuctionTrigger = 0
	// Batch auction
	AuctionTrigger_AUCTION_TRIGGER_BATCH AuctionTrigger = 1
	// Opening auction
	AuctionTrigger_AUCTION_TRIGGER_OPENING AuctionTrigger = 2
	// Price monitoring trigger
	AuctionTrigger_AUCTION_TRIGGER_PRICE AuctionTrigger = 3
	// Liquidity monitoring trigger
	AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY AuctionTrigger = 4
)
