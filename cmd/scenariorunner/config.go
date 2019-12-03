package main

import (
	"time"

	"code.vegaprotocol.io/vega/cmd/scenariorunner/core"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func NewDefaultConfig() core.Config {
	return core.Config{
		InitialTime:                 &timestamp.Timestamp{Seconds: 1546416000, Nanos: 0}, //Corresponds to 2/1/2019 8:00am UTC
		AdvanceTimeAfterInstruction: true,
		TimeDelta:                   ptypes.DurationProto(time.Nanosecond),
		OmitUnsupportedInstructions: true,
		OmitInvalidInstructions:     true,
		Markets: []*types.Market{
			{
				Id:   "JXGQYDVQAP5DJUAQBCB4PACVJPFJR4XI",
				Name: "ETHBTC/DEC19",
				TradableInstrument: &types.TradableInstrument{
					Instrument: &types.Instrument{
						Id:        "Crypto/ETHBTC/Futures/Dec19",
						Code:      "CRYPTO:ETHBTC/DEC19",
						Name:      "December 2019 ETH vs BTC future",
						BaseName:  "ETH",
						QuoteName: "BTC",
						Metadata: &types.InstrumentMetadata{
							Tags: []string{"asset_class:fx/crypto",
								"product:futures"},
						},
						InitialMarkPrice: 5,
						Product: &types.Instrument_Future{
							Future: &types.Future{
								Maturity: "2019-12-31T23:59:59Z",
								Asset:    "BTC",
								Oracle: &types.Future_EthereumEvent{
									EthereumEvent: &types.EthereumEvent{
										ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
										Event:      "price_changed",
									},
								},
							},
						},
					},
					MarginCalculator: &types.MarginCalculator{
						ScalingFactors: &types.ScalingFactors{
							SearchLevel:       1.1,
							InitialMargin:     1.2,
							CollateralRelease: 1.4,
						},
					},
					RiskModel: &types.TradableInstrument_ForwardRiskModel{
						ForwardRiskModel: &types.ForwardRiskModel{
							RiskAversionParameter: 0.01,
							Tau:                   0.00011407711613050422,
							Params: &types.ModelParamsBS{
								R:     0.016,
								Sigma: 0.09,
							},
						},
					},
				},
				DecimalPlaces: 5,
				TradingMode: &types.Market_Continuous{
					Continuous: &types.ContinuousTrading{},
				},
			},
		},
	}
}
