package scenariorunner

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

//ErrNotImplemented throws an error with "NotImplemented" text
var ErrNotImplemented error = errors.New("NotImplemented")

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

	executionEngine := execution.NewEngine(
		log,
		config.Execution,
		vegatime.New(config.Time),
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
		res, err := sr.processInstruction(instr)
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

func (sr ScenarioRunner) processInstruction(instr *Instruction) (*InstructionResult, error) {

	var err error
	var responseErr error
	var response proto.Message
	switch strings.ToLower(instr.Request) {
	case "notifytraderaccount":
		req := &types.NotifyTraderAccount{}
		err = proto.Unmarshal(instr.Message.Value, req)
		if err != nil {
			break
		}
		responseErr = sr.executionEngine.NotifyTraderAccount(req)
		response = nil
	case "submitorder":
		req := &types.Order{}
		//err = ptypes.UnmarshalAny(instr.Message, req)
		err = proto.Unmarshal(instr.Message.Value, req)
		if err != nil {
			break
		}
		response, responseErr = sr.executionEngine.SubmitOrder(req)
	default:
		return nil, fmt.Errorf("Unsupported request: %v", instr.Request)
	}
	return instr.NewResult(response, responseErr)

}
