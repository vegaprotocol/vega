package scenariorunner

import (
	"context"
	"errors"
	"strings"
	"time"

	cfg "code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/scenariorunner/preprocessors"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/hashicorp/go-multierror"
)

var (
	ErrNotImplemented       error = errors.New("Not implemented")
	ErrDuplicateInstruction error = errors.New("Duplicate instruction")
)

type ScenarioRunner struct {
	Config       Config
	timeProvider *preprocessors.Time
	providers    []core.PreProcessorProvider
}

// NewScenarioRunner returns a pointer to new instance of scenario runner
func NewScenarioRunner() (*ScenarioRunner, error) {

	d, err := getDependencies()
	if err != nil {
		return nil, err
	}
	execution := preprocessors.NewExecution(d.execution)
	marketDepth := preprocessors.NewMarketDepth(d.ctx, d.market, d.trade)
	markets := preprocessors.NewMarkets(d.ctx, d.market)
	orders := preprocessors.NewOrders(d.ctx, d.order)
	trades := preprocessors.NewTrades(d.ctx, d.trade)

	time := preprocessors.NewTime(d.vegaTime)

	return &ScenarioRunner{
		Config:       NewDefaultConfig(),
		timeProvider: time,
		providers: []core.PreProcessorProvider{
			execution,
			marketDepth,
			markets,
			orders,
			trades,
		},
	}, nil
}

func (sr ScenarioRunner) flattenPreProcessors() (map[string]*core.PreProcessor, error) {
	maps := make(map[string]*core.PreProcessor)
	for _, provider := range append(sr.providers, sr.timeProvider) {
		m := provider.PreProcessors()
		for k, v := range m {
			if _, ok := maps[k]; ok {
				return nil, ErrDuplicateInstruction
			}
			maps[k] = v
		}
	}
	return maps, nil
}

// ProcessInstructions takes a set of instructions and submits them to the protocol
func (sr ScenarioRunner) ProcessInstructions(instrSet core.InstructionSet) (*core.ResultSet, error) {
	var processed, omitted uint64
	n := len(instrSet.Instructions)
	results := make([]*core.InstructionResult, n)
	var errors *multierror.Error

	preProcessors, err := sr.flattenPreProcessors()
	if err != nil {
		return nil, err
	}

	for i, instr := range instrSet.Instructions {
		// TODO (WG 01/11/2019) matching by lower case by convention only, enforce with a custom type
		preProcessor, ok := preProcessors[strings.ToLower(instr.Request)]
		if !ok {
			if !sr.Config.OmitUnsupportedInstructions {
				return nil, errors.ErrorOrNil()
			}
			errors = multierror.Append(errors, core.ErrInstructionNotSupported)
			omitted++
			continue
		}
		p, err := preProcessor.PreProcess(instr)
		if err != nil {
			if !sr.Config.OmitInvalidInstructions {
				return nil, errors.ErrorOrNil()
			}
			errors = multierror.Append(errors, err)
			omitted++
			continue
		}
		res, err := p.Result()
		if err != nil {
			if !sr.Config.OmitInvalidInstructions {
				return nil, errors.ErrorOrNil()
			}
			errors = multierror.Append(errors, err)
			omitted++
			continue
		}
		results[i] = res
		processed++
		if sr.Config.AdvanceTimeAfterInstruction {
			err := sr.timeProvider.AdvanceTime(sr.Config.AdvanceDuration)
			if err != nil {
				return nil, err
			}
		}

	}

	md := &core.Metadata{
		InstructionsProcessed: processed,
		InstructionsOmitted:   omitted,
	}

	return &core.ResultSet{
		Summary: md,
		Results: results,
	}, errors.ErrorOrNil()
}

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

	orderStore, err := storage.NewOrders(log, config.Storage, cancel)
	if err != nil {
		return nil, err
	}
	tradeStore, err := storage.NewTrades(log, config.Storage, cancel)
	if err != nil {
		return nil, err
	}
	riskStore, err := storage.NewRisks(config.Storage)
	if err != nil {
		return nil, err
	}
	candleStore, err := storage.NewCandles(log, config.Storage)
	if err != nil {
		return nil, err
	}

	marketStore, err := storage.NewMarkets(log, config.Storage)
	if err != nil {
		return nil, err
	}

	partyStore, err := storage.NewParties(config.Storage)
	if err != nil {
		return nil, err
	}

	accounts, err := storage.NewAccounts(log, config.Storage)
	if err != nil {
		return nil, err
	}

	transferResponseStore, err := storage.NewTransferResponses(log, config.Storage)
	if err != nil {
		return nil, err
	}

	marketService, err := markets.NewService(log, config.Markets, marketStore, orderStore)
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
		accounts,
		transferResponseStore,
	)

	tradeService, err := trades.NewService(log, config.Trades, tradeStore, riskStore)

	return &dependencies{
		ctx:       ctx,
		vegaTime:  timeService,
		execution: engine,
		order:     orderStore,
		trade:     tradeService,
		market:    marketService,
	}, nil
}

type dependencies struct {
	ctx       context.Context
	vegaTime  *vegatime.Svc
	execution *execution.Engine
	order     *storage.Order
	trade     *trades.Svc
	market    *markets.Svc
}
