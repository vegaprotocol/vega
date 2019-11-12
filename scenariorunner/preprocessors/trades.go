package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
)

type Trades struct {
	ctx        context.Context
	tradeStore *storage.Trade
	tdp        api.TradeDataProvider
}

func NewTrades(ctx context.Context, tradeStore *storage.Trade, tdp api.TradeDataProvider) *Trades {
	return &Trades{ctx, tradeStore, tdp}
}

func (t *Trades) PreProcessors() map[string]*core.PreProcessor {
	return map[string]*core.PreProcessor{
		"tradesbymarket": t.tradesByMarket(),
		"tradesbyparty":  t.tradesByParty(),
		"tradesbyorder":  t.tradesByOrder(),
		"lasttrade":      t.lastTrade(),
	}
}

func (t *Trades) tradesByMarket() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.TradesByMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				t.commitStore()
				return api.ProcessTradesByMarket(t.ctx, req, t.tdp)
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.TradesByMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) tradesByParty() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.TradesByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				t.commitStore()
				return api.ProcessTradesByParty(t.ctx, req, t.tdp)
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.TradesByPartyRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) tradesByOrder() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.TradesByOrderRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				t.commitStore()
				return api.ProcessTradesByOrder(t.ctx, req, t.tdp)
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.TradesByOrderRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) lastTrade() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.LastTradeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessLastTrade(t.ctx, req, t.tdp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.LastTradeRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) commitStore() {
	t.tradeStore.Commit()
}
