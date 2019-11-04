package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

type Markets struct {
	mappings map[string]*core.PreProcessor
}

func NewMarkets(ctx context.Context, mdp api.MarketDataProvider) *Markets {

	m := map[string]*core.PreProcessor{
		"marketbyid": marketByID(ctx, mdp),
		"markets":    markets(ctx, mdp),
	}

	return &Markets{m}
}

func (m *Markets) PreProcessors() map[string]*core.PreProcessor {
	return m.mappings
}

func marketByID(ctx context.Context, mdp api.MarketDataProvider) *core.PreProcessor {
	req := &protoapi.MarketByIDRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarketByID(ctx, req, mdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func markets(ctx context.Context, mdp api.MarketDataProvider) *core.PreProcessor {
	req := &empty.Empty{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarkets(ctx, req, mdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
