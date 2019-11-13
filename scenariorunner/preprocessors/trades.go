package preprocessors

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
)

type Trades struct {
	ctx        context.Context
	tradeStore *storage.Trade
}

func NewTrades(ctx context.Context, tradeStore *storage.Trade) *Trades {
	return &Trades{ctx, tradeStore}
}

func (t *Trades) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_TRADES_BY_MARKET: t.tradesByMarket(),
		core.RequestType_TRADES_BY_PARTY:  t.tradesByParty(),
		core.RequestType_TRADES_BY_ORDER:  t.tradesByOrder(),
		core.RequestType_LAST_TRADE:       t.lastTrade(),
	}
}

func (t *Trades) tradesByMarket() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.TradesByMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		pagination := core.GetDefaultPagination(req.Pagination)
		return instr.PreProcess(
			func() (proto.Message, error) {
				t.commitStore()
				resp, err := t.tradeStore.GetByMarket(t.ctx, req.MarketID, pagination.Skip, pagination.Limit, pagination.Descending)
				if err != nil {
					return nil, err
				}
				return &protoapi.TradesByMarketResponse{Trades: resp}, nil
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
		pagination := core.GetDefaultPagination(req.Pagination)
		return instr.PreProcess(
			func() (proto.Message, error) {
				t.commitStore()
				resp, err := t.tradeStore.GetByParty(t.ctx, req.PartyID, pagination.Skip, pagination.Limit, pagination.Descending, &req.MarketID)
				if err != nil {
					return nil, err
				}
				return &protoapi.TradesByPartyResponse{Trades: resp}, nil
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
				resp, err := t.tradeStore.GetByOrderID(t.ctx, req.OrderID, 0, 0, false, nil)
				if err != nil {
					return nil, err
				}
				return &protoapi.TradesByOrderResponse{Trades: resp}, nil

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
			func() (proto.Message, error) {
				t.commitStore()
				resp, err := t.tradeStore.GetByMarket(t.ctx, req.MarketID, 0, 1, true)
				if err != nil {
					return nil, err
				}
				return &protoapi.LastTradeResponse{Trade: resp[0]}, nil

			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.LastTradeRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) commitStore() {
	t.tradeStore.Commit()
}
