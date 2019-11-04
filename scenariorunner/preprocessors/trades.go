package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
)

type Trades struct {
	mappings map[string]*core.PreProcessor
}

func NewTrades(ctx context.Context, tdp api.TradeDataProvider) *Trades {

	m := map[string]*core.PreProcessor{
		"tradesbymarket": tradesByMarket(ctx, tdp),
		"tradesbyparty":  tradesByParty(ctx, tdp),
		"tradesbyorder":  tradesByOrder(ctx, tdp),
		"lasttrade":      lastTrade(ctx, tdp),
	}

	return &Trades{m}
}

func (t *Trades) PreProcessors() map[string]*core.PreProcessor {
	return t.mappings
}

func tradesByMarket(ctx context.Context, tdp api.TradeDataProvider) *core.PreProcessor {
	req := &protoapi.TradesByMarketRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessTradesByMarket(ctx, req, tdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func tradesByParty(ctx context.Context, tdp api.TradeDataProvider) *core.PreProcessor {
	req := &protoapi.TradesByPartyRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessTradesByParty(ctx, req, tdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func tradesByOrder(ctx context.Context, tdp api.TradeDataProvider) *core.PreProcessor {
	req := &protoapi.TradesByOrderRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessTradesByOrder(ctx, req, tdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func lastTrade(ctx context.Context, tdp api.TradeDataProvider) *core.PreProcessor {
	req := &protoapi.LastTradeRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessLastTrade(ctx, req, tdp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
