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
	ctx         context.Context
	marketStore *storage.Market
	orderStore  *storage.Order
	mdp         api.MarketDataProvider
	tdp         api.TradeDataProvider
}

func NewMarkets(ctx context.Context, marketStore *storage.Market, orderStore *storage.Order, mdp api.MarketDataProvider, tdp api.TradeDataProvider) *Markets {
	return &Markets{ctx, marketStore, orderStore, mdp, tdp}
}

func (m *Markets) PreProcessors() map[string]*core.PreProcessor {
	return map[string]*core.PreProcessor{
		"marketbyid":  m.marketByID(),
		"markets":     m.markets(),
		"marketdepth": m.marketDepth(),
	}
}

func (m *Markets) marketByID() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.MarketByIDRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarketByID(m.ctx, req, m.mdp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.MarketByIDRequest{},
		PreProcess:   preProcessor,
	}
}

func (m *Markets) markets() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &empty.Empty{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarkets(m.ctx, req, m.mdp) })
	}
	return &core.PreProcessor{
		MessageShape: &empty.Empty{},
		PreProcess:   preProcessor,
	}
}

func (m *Markets) marketDepth() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.MarketDepthRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessMarketDepth(m.ctx, req, m.mdp, m.tdp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.MarketDepthRequest{},
		PreProcess:   preProcessor,
	}
}
