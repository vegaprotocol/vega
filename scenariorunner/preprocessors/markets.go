package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

type Markets struct {
	marketStore *storage.Market
	orderStore  *storage.Order
	mappings    map[string]*core.PreProcessor
}

func NewMarkets(ctx context.Context, marketStore *storage.Market, orderStore *storage.Order, mdp api.MarketDataProvider, tdp api.TradeDataProvider) *Markets {

	m := map[string]*core.PreProcessor{
		"marketbyid":  marketByID(ctx, mdp),
		"markets":     markets(ctx, mdp),
		"marketdepth": marketDepth(ctx, mdp, tdp),
	}

	return &Markets{marketStore, orderStore, m}
}

func (m *Markets) PreProcessors() map[string]*core.PreProcessor {
	return m.mappings
}

func marketByID(ctx context.Context, mdp api.MarketDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.MarketByIDRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarketByID(ctx, req, mdp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.MarketByIDRequest{},
		PreProcess:   preProcessor,
	}
}

func markets(ctx context.Context, mdp api.MarketDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &empty.Empty{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarkets(ctx, req, mdp) })
	}
	return &core.PreProcessor{
		MessageShape: &empty.Empty{},
		PreProcess:   preProcessor,
	}
}

func marketDepth(ctx context.Context, mdp api.MarketDataProvider, tdp api.TradeDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.MarketDepthRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarketDepth(ctx, req, mdp, tdp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.MarketDepthRequest{},
		PreProcess:   preProcessor,
	}
}
