package scenariorunner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
	//"github.com/golang/protobuf/ptypes"
	"github.com/hashicorp/go-multierror"
)

var (
	ErrNotImplemented          error = errors.New("Not implemented")
	ErrInstructionNotSupported error = errors.New("Instruction not supported")
	ErrInstructionInvalid      error = errors.New("Instruction invalid")
)

type ScenarioRunner struct {
	executionEngine *execution.Engine
}

// NewScearioRunner returns a pointer to new instance of scenario runner
func NewScenarionRunner() (*ScenarioRunner, error) {
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
	executionEngine := execution.NewEngine(
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

	return &ScenarioRunner{executionEngine}, nil
}

// ProcessInstructions takes a set of instructions and submits them to the protocol
func (sr ScenarioRunner) ProcessInstructions(instrSet InstructionSet) (ResultSet, error) {
	var processed, omitted uint64
	n := len(instrSet.Instructions)
	results := make([]*InstructionResult, n)
	var errors *multierror.Error

	for i, instr := range instrSet.Instructions {
		p, err := sr.preProcess(instr)
		if err != nil {
			errors = multierror.Append(errors, err)
			omitted++
			continue
		}
		res, err := p.result()
		if err != nil {
			errors = multierror.Append(errors, err)
			omitted++
			continue
		}
		results[i] = res
		processed++
	}

	md := &Metadata{
		InstructionsProcessed: processed,
		InstructionsOmitted:   omitted,
	}

	return ResultSet{
		Summary: md,
		Results: results,
	}, errors.ErrorOrNil()
}

func (sr ScenarioRunner) preProcess(instr *Instruction) (*preProcessedInstruction, error) {
	switch strings.ToLower(instr.Request) {
	case "notifytraderaccount":
		req := &types.NotifyTraderAccount{}
		err := proto.Unmarshal(instr.Message.Value, req)
		if err != nil {
			return nil, ErrInstructionInvalid
		}
		return &preProcessedInstruction{
			instruction: instr,
			payload:     req,
			deliver:     func() (proto.Message, error) { return nil, sr.executionEngine.NotifyTraderAccount(req) },
		}, nil
	case "submitorder":
		req := &types.Order{}
		err := proto.Unmarshal(instr.Message.Value, req)
		if err != nil {
			return nil, ErrInstructionInvalid
		}
		return &preProcessedInstruction{
			instruction: instr,
			payload:     req,
			deliver:     func() (proto.Message, error) { return sr.executionEngine.SubmitOrder(req) },
		}, nil
	default:
		return nil, fmt.Errorf("Unsupported request: %v", instr.Request)
	}
}

type preProcessedInstruction struct {
	instruction *Instruction
	payload     proto.Message
	deliver     func() (proto.Message, error)
}

func (p *preProcessedInstruction) result() (*InstructionResult, error) {
	return p.instruction.NewResult(p.deliver())
}
