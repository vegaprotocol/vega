package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
)

type MarketDepth struct {
	mappings map[string]*core.PreProcessor
}

func NewMarketDepth(ctx context.Context, mdp api.MarketDataProvider, tdp api.TradeDataProvider) *MarketDepth {

	m := map[string]*core.PreProcessor{
		"marketdepth": marketDepth(ctx, mdp, tdp),
	}

	return &MarketDepth{m}
}

func (m *MarketDepth) PreProcessors() map[string]*core.PreProcessor {
	return m.mappings
}

func marketDepth(ctx context.Context, mdp api.MarketDataProvider, tdp api.TradeDataProvider) *core.PreProcessor {
	req := &protoapi.MarketDepthRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarketDepth(ctx, req, mdp, tdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
