//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

type Market = proto.Market
type MarketData = proto.MarketData
type Instrument = proto.Instrument
type Instrument_Future = proto.Instrument_Future
type Future = proto.Future
type Market_Continuous = proto.Market_Continuous
type Market_Discrete = proto.Market_Discrete
type TradableInstrument = proto.TradableInstrument
type LiquidityProviderFeeShare = proto.LiquidityProviderFeeShare

type MarketTimestamps struct {
	Proposed int64
	Pending  int64
	Open     int64
	Close    int64
}

func (m MarketTimestamps) IntoProto() *proto.MarketTimestamps {
	return &proto.MarketTimestamps{
		Proposed: m.Proposed,
		Pending:  m.Pending,
		Open:     m.Open,
		Close:    m.Close,
	}
}

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

type InstrumentMetadata struct {
	Tags []string
}

func (i InstrumentMetadata) IntoProto() *proto.InstrumentMetadata {
	tags := make([]string, 0, len(i.Tags))
	return &proto.InstrumentMetadata{
		Tags: append(tags, i.Tags...),
	}
}

func (i InstrumentMetadata) String() string {
	return i.IntoProto().String()
}

type Timestamp struct {
	Value int64
}

type Price struct {
	Value *num.Uint
}

type AuctionDuration struct {
	Duration int64
	Volume   uint64
}

func (a AuctionDuration) IntoProto() *proto.AuctionDuration {
	return &proto.AuctionDuration{
		Duration: a.Duration,
		Volume:   a.Volume,
	}
}

func (a AuctionDuration) String() string {
	return a.IntoProto().String()
}

func (p Price) IntoProto() *proto.Price {
	return &proto.Price{
		Value: p.Value.Uint64(),
	}
}

func (p Price) String() string {
	return p.IntoProto().String()
}

func (t Timestamp) IntoProto() *proto.Timestamp {
	return &proto.Timestamp{
		Value: t.Value,
	}
}

func (t Timestamp) String() string {
	return t.IntoProto().String()
}
