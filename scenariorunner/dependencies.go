package scenariorunner

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hashicorp/go-multierror"
)

//TODO (WG 05/11/2019): instantiating dependencies internally while WIP, the final dependencies will get incjeted from outside the package.
func getDependencies(log *logging.Logger, config storage.Config) (*dependencies, error) {

	ctx, cancel := context.WithCancel(context.Background())
	var errs *multierror.Error

	orderStore, err := storage.NewOrders(log, config, cancel)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	tradeStore, err := storage.NewTrades(log, config, cancel)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	candleStore, err := storage.NewCandles(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	marketStore, err := storage.NewMarkets(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	partyStore, err := storage.NewParties(config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	accountStore, err := storage.NewAccounts(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	transferResponseStore, err := storage.NewTransferResponses(log, config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	riskStore, err := storage.NewRisks(config)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	tradeService, err := trades.NewService(log, trades.NewDefaultConfig(), tradeStore, riskStore)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = errs.ErrorOrNil()
	if err != nil {
		return nil, err
	}

	executionConfig := execution.NewDefaultConfig("")
	timeService := vegatime.New(vegatime.NewDefaultConfig())
	engine := execution.NewEngine(
		log,
		executionConfig,
		timeService,
		orderStore,
		tradeStore,
		candleStore,
		marketStore,
		partyStore,
		accountStore,
		transferResponseStore,
	)

	return &dependencies{
		ctx:          ctx,
		vegaTime:     timeService,
		execution:    engine,
		partyStore:   partyStore,
		orderStore:   orderStore,
		tradeStore:   tradeStore,
		marketStore:  marketStore,
		accountStore: accountStore,
		candleStore:  candleStore,
		tradeService: tradeService,
	}, nil
}

type dependencies struct {
	ctx          context.Context
	vegaTime     *vegatime.Svc
	execution    *execution.Engine
	partyStore   *storage.Party
	orderStore   *storage.Order
	tradeStore   *storage.Trade
	marketStore  *storage.Market
	accountStore *storage.Account
	candleStore  *storage.Candle
	tradeService *trades.Svc
}

func NewDefaultConfig() core.Config {
	return core.Config{
		InitialTime:                 &timestamp.Timestamp{Seconds: 1546416000, Nanos: 0}, //Corresponds to 2/1/2019 8:00am UTC
		AdvanceTimeAfterInstruction: true,
		TimeDelta:                   ptypes.DurationProto(time.Nanosecond),
		OmitUnsupportedInstructions: true,
		OmitInvalidInstructions:     true,
		Markets: []*types.Market{
			&types.Market{
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
