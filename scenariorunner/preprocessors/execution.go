package preprocessors

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
)

type Execution struct {
	mappings map[string]*core.PreProcessor
}

func NewExecution() (*Execution, error) {
	e, err := newExecutionEngine()
	if err != nil {
		return nil, err
	}
	m := map[string]*core.PreProcessor{
		"notifytraderaccount": notifyTraderAccount(e),
		"submitorder":         submitOrder(e),
		"cancelorder":         cancelOrder(e),
		"amendorder":          amendOrder(e),
		"withdraw":            withdraw(e),
	}

	return &Execution{m}, nil
}

func (e *Execution) PreProcessors() map[string]*core.PreProcessor {
	return e.mappings
}

func notifyTraderAccount(e *execution.Engine) *core.PreProcessor {
	req := &types.NotifyTraderAccount{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return nil, e.NotifyTraderAccount(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func submitOrder(e *execution.Engine) *core.PreProcessor {
	req := &types.Order{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(

			func() (proto.Message, error) { return e.SubmitOrder(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func cancelOrder(e *execution.Engine) *core.PreProcessor {
	req := &types.Order{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return e.CancelOrder(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func amendOrder(e *execution.Engine) *core.PreProcessor {
	req := &types.OrderAmendment{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return e.AmendOrder(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func withdraw(e *execution.Engine) *core.PreProcessor {
	req := &types.Withdraw{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return nil, e.Withdraw(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func newExecutionEngine() (*execution.Engine, error) {
	log := logging.NewDevLogger()
	log.SetLevel(logging.InfoLevel)

	ctx, cancel := context.WithCancel(context.Background())
	configPath := fsutil.DefaultVegaDir()
	cfgwatchr, err := config.NewFromFile(ctx, log, configPath, configPath)
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

	return engine, nil
}
