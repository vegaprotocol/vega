package main

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/cmd/scenariorunner/core"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/ptypes"
	"github.com/hashicorp/go-multierror"
)

var (
	ErrDuplicateInstruction error = errors.New("duplicate instruction")
)

type Engine struct {
	Config           core.Config
	Version          string
	summaryGenerator *core.SummaryGenerator
	timeControl      *core.TimeControl
	providers        []core.PreProcessorProvider
	tradesGenerated  uint64
	Execution        *execution.Engine
}

// NewEngine returns a pointer to new instance of scenario runner
func NewEngine(log *logging.Logger, engineConfig core.Config, storageConfig storage.Config, version string) (*Engine, error) {

	d, err := getDependencies(log, storageConfig)
	if err != nil {
		return nil, err
	}
	timeControl := core.NewTimeControl(d.vegaTime)
	time := core.NewTime(timeControl)
	initialTime, err := ptypes.Timestamp(engineConfig.InitialTime)
	if err != nil {
		return nil, err
	}
	timeControl.SetTime(initialTime)

	for _, mkt := range engineConfig.Markets {
		err = d.execution.SubmitMarket(mkt)
		if err != nil {
			return nil, err
		}
	}

	execution := core.NewExecution(d.execution)

	markets := core.NewMarkets(d.ctx, d.marketStore)
	orders := core.NewOrders(d.ctx, d.orderStore)
	trades := core.NewTrades(d.ctx, d.tradeStore)
	accounts := core.NewAccounts(d.ctx, d.accountStore)
	candles := core.NewCandles(d.ctx, d.candleStore)
	positions := core.NewPositions(d.ctx, d.tradeService)
	parties := core.NewParties(d.ctx, d.partyStore)

	summaryGenerator := core.NewSummaryGenerator(d.ctx, d.tradeStore, d.orderStore, d.partyStore, d.marketStore, d.accountStore, d.tradeService, d.execution)
	summary := core.NewSummary(summaryGenerator)

	err = d.execution.Generate()
	if err != nil {
		return nil, err
	}

	return &Engine{
		Config:           engineConfig,
		Version:          version,
		summaryGenerator: summaryGenerator,
		timeControl:      timeControl,
		providers: []core.PreProcessorProvider{
			execution,
			markets,
			orders,
			trades,
			accounts,
			candles,
			positions,
			parties,
			summary,
			time,
		},
		Execution: d.execution,
	}, nil
}

// ProcessInstructions takes a set of instructions and submits them to the protocol
func (e *Engine) ProcessInstructions(instrSet core.InstructionSet) (*core.ResultSet, error) {
	start := time.Now()
	var processed, omitted uint64
	results := make([]*core.InstructionResult, len(instrSet.Instructions))
	var errs *multierror.Error

	preProcessors, err := e.flattenPreProcessors()
	if err != nil {
		return nil, err
	}

	initialState, err := e.summaryGenerator.Summary(nil)
	if err != nil {
		return nil, err
	}

	duration, err := ptypes.Duration(e.Config.TimeDelta)
	if err != nil {
		return nil, err
	}
	//TODO (WG 08/11/2019): Split into 3 separate loops (check if instruction supported, check if instructions valid, check if instruction processed w/o errors) to fail early
	for i, instr := range instrSet.Instructions {
		preProcessor, ok := preProcessors[instr.Request]
		if !ok {
			if !e.Config.OmitUnsupportedInstructions {
				return nil, errs.ErrorOrNil()
			}
			errs = multierror.Append(errs, core.ErrInstructionNotSupported)
			omitted++
			continue
		}
		p, err := preProcessor.PreProcess(instr)
		if err != nil {
			if !e.Config.OmitInvalidInstructions {
				return nil, errs.ErrorOrNil()
			}
			errs = multierror.Append(errs, err)
			omitted++
			continue
		}
		res, err := p.Result()
		if err != nil {
			if !e.Config.OmitInvalidInstructions {
				return nil, errs.ErrorOrNil()
			}
			errs = multierror.Append(errs, err)
			omitted++
			continue
		}
		if len(res.Error) > 0 {
			fmt.Println("ERROR: " + res.Error)
		}
		results[i] = res
		processed++
		if e.Config.AdvanceTimeAfterInstruction {
			err := e.timeControl.AdvanceTime(duration)
			if err != nil {
				return nil, err
			}
		}
		err = e.Execution.Generate()
		if err != nil {
			return nil, err
		}
	}
	finalState, err := e.summaryGenerator.Summary(nil)
	if err != nil {
		return nil, err
	}
	summary, err := e.ExtractData()
	if err != nil {
		return nil, err
	}

	totalTrades := sumTrades(*summary)

	md := &core.Metadata{
		InstructionsProcessed: processed,
		InstructionsOmitted:   omitted,
		TradesGenerated:       totalTrades - e.tradesGenerated,
		FinalMarketDepth:      marketDepths(*summary),
		ProcessingTime:        ptypes.DurationProto(time.Since(start)),
	}

	e.tradesGenerated = totalTrades

	return &core.ResultSet{
		Metadata:     md,
		Results:      results,
		InitialState: initialState.Summary,
		FinalState:   finalState.Summary,
		Config:       &e.Config,
		Version:      e.Version,
	}, errs.ErrorOrNil()
}

func (e Engine) ExtractData() (*core.SummaryResponse, error) {
	return e.summaryGenerator.Summary(nil)
}

func sumTrades(response core.SummaryResponse) uint64 {
	var trades int
	for _, mkt := range response.Summary.Markets {
		if mkt != nil {
			trades += +len(mkt.Trades)
		}

	}

	return uint64(trades)
}

func marketDepths(response core.SummaryResponse) []*proto.MarketDepth {
	d := make([]*proto.MarketDepth, len(response.Summary.Markets))
	for i, mkt := range response.Summary.Markets {
		if mkt != nil {
			d[i] = mkt.MarketDepth
		}
	}
	return d
}

func (e *Engine) flattenPreProcessors() (map[core.RequestType]*core.PreProcessor, error) {
	maps := make(map[core.RequestType]*core.PreProcessor)
	for _, provider := range e.providers {
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
