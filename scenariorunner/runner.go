package scenariorunner

import (
	"errors"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/scenariorunner/preprocessors"

	"github.com/golang/protobuf/ptypes"
	"github.com/hashicorp/go-multierror"
)

var (
	ErrNotImplemented       error = errors.New("Not implemented")
	ErrDuplicateInstruction error = errors.New("Duplicate instruction")
)

type ScenarioRunner struct {
	Config           Config
	summaryGenerator *core.SummaryGenerator
	internalProvider *internalProvider
	providers        []core.PreProcessorProvider
	tradesGenerated  uint64
}

// NewScenarioRunner returns a pointer to new instance of scenario runner
func NewScenarioRunner() (*ScenarioRunner, error) {

	d, err := getDependencies()
	if err != nil {
		return nil, err
	}
	execution := preprocessors.NewExecution(d.execution)
	marketDepth := preprocessors.NewMarketDepth(d.ctx, d.marketService, d.tradeService)
	markets := preprocessors.NewMarkets(d.ctx, d.marketService)
	orders := preprocessors.NewOrders(d.ctx, d.orderStore)
	trades := preprocessors.NewTrades(d.ctx, d.tradeService)

	summaryGenerator := core.NewSummaryGenerator(d.ctx, d.marketService, d.tradeStore, d.orderStore, d.partyStore)

	internal := newInternalProvider(d.vegaTime, summaryGenerator)

	return &ScenarioRunner{
		Config:           NewDefaultConfig(),
		summaryGenerator: summaryGenerator,
		internalProvider: internal,
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
	for _, provider := range append(sr.providers, sr.internalProvider) {
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
	start := time.Now()
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
			err := sr.internalProvider.AdvanceTime(sr.Config.AdvanceDuration)
			if err != nil {
				return nil, err
			}
		}

	}

	summary, err := sr.summaryGenerator.ProtocolSummary(nil)
	if err != nil {
		return nil, err
	}

	totalTrades := sumTrades(*summary)

	md := &core.Metadata{
		InstructionsProcessed: processed,
		InstructionsOmitted:   omitted,
		TradesGenerated:       totalTrades - sr.tradesGenerated,
		FinalMarketDepth:      marketDepths(*summary),
		ProcessingTime:        ptypes.DurationProto(time.Since(start)),
	}

	sr.tradesGenerated = totalTrades

	return &core.ResultSet{
		Metadata: md,
		Results:  results,
	}, errors.ErrorOrNil()
}

func sumTrades(summary core.ProtocolSummaryResponse) uint64 {
	var trades int
	for _, mkt := range summary.Markets {
		if mkt != nil {
			trades += +len(mkt.Trades)
		}

	}

	return uint64(trades)
}

func marketDepths(summary core.ProtocolSummaryResponse) []*proto.MarketDepth {
	d := make([]*proto.MarketDepth, len(summary.Markets))
	for i, mkt := range summary.Markets {
		if mkt != nil {
			d[i] = mkt.MarketDepth
		}
	}
	return d
}
