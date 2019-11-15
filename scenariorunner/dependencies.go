package scenariorunner

import (
	"context"
	"time"

	cfg "code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hashicorp/go-multierror"
)

//TODO (WG 05/11/2019): instantiating dependencies internally while WIP, the final dependencies will get incjeted from outside the package.
func getDependencies() (*dependencies, error) {
	log := logging.NewDevLogger()
	log.SetLevel(logging.InfoLevel)

	ctx, cancel := context.WithCancel(context.Background())
	configPath := fsutil.DefaultVegaDir()
	cfgwatchr, err := cfg.NewFromFile(ctx, log, configPath, configPath)
	if err != nil {
		log.Error("unable to start config watcher", logging.Error(err))
		cancel()
		return nil, err
	}
	config := cfgwatchr.Get()
	log = logging.NewLoggerFromConfig(config.Logging)

	var errs *multierror.Error

	orderStore, err := storage.NewOrders(log, config.Storage, cancel)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	tradeStore, err := storage.NewTrades(log, config.Storage, cancel)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	candleStore, err := storage.NewCandles(log, config.Storage)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	marketStore, err := storage.NewMarkets(log, config.Storage)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	partyStore, err := storage.NewParties(config.Storage)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	accountStore, err := storage.NewAccounts(log, config.Storage)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	transferResponseStore, err := storage.NewTransferResponses(log, config.Storage)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	riskStore, err := storage.NewRisks(config.Storage)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	tradeService, err := trades.NewService(log, config.Trades, tradeStore, riskStore)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = errs.ErrorOrNil()
	if err != nil {
		return nil, err
	}

	timeService := vegatime.New(config.Time)
	now := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	timeService.SetTimeNow(now)
	engine := execution.NewEngine(
		log,
		config.Execution,
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
		ProtocolTime:                &timestamp.Timestamp{Seconds: 1546416000, Nanos: 0}, //Corresponds to 2/1/2019 8:00am UTC
		AdvanceTimeAfterInstruction: true,
		TimeDelta:                   ptypes.DurationProto(time.Nanosecond),
		OmitUnsupportedInstructions: true,
		OmitInvalidInstructions:     true,
	}
}
