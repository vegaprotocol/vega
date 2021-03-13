//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import "code.vegaprotocol.io/vega/proto"

type Market = proto.Market
type MarketData = proto.MarketData
type Instrument = proto.Instrument
type Instrument_Future = proto.Instrument_Future
type Future = proto.Future
type TradableInstrument = proto.TradableInstrument
type TradableInstrument_LogNormalRiskModel = proto.TradableInstrument_LogNormalRiskModel
type TradableInstrument_SimpleRiskModel = proto.TradableInstrument_SimpleRiskModel
