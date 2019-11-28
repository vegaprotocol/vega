package core

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
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

func (t *Trades) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_TRADES_BY_MARKET: t.tradesByMarket(),
		RequestType_TRADES_BY_PARTY:  t.tradesByParty(),
		RequestType_TRADES_BY_ORDER:  t.tradesByOrder(),
		RequestType_LAST_TRADE:       t.lastTrade(),
	}
}

func (t *Trades) tradesByMarket() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.TradesByMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		pagination := GetDefaultPagination(req.Pagination)
		return instr.PreProcess(
			func() (proto.Message, error) {
				//t.commitStore()
				resp, err := t.tradeStore.GetByMarket(t.ctx, req.MarketID, pagination.Skip, pagination.Limit, pagination.Descending)
				if err != nil {
					return nil, err
				}
				return &protoapi.TradesResponse{Trades: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.TradesByMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) tradesByParty() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.TradesByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		pagination := GetDefaultPagination(req.Pagination)
		return instr.PreProcess(
			func() (proto.Message, error) {
				//t.commitStore()
				resp, err := t.tradeStore.GetByParty(t.ctx, req.PartyID, pagination.Skip, pagination.Limit, pagination.Descending, &req.MarketID)
				if err != nil {
					return nil, err
				}
				return &protoapi.TradesResponse{Trades: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.TradesByPartyRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) tradesByOrder() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.TradesByOrderRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				//t.commitStore()
				resp, err := t.tradeStore.GetByOrderID(t.ctx, req.OrderID, 0, 0, false, nil)
				if err != nil {
					return nil, err
				}
				return &protoapi.TradesResponse{Trades: resp}, nil

			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.TradesByOrderRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Trades) lastTrade() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.LastTradeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				//t.commitStore()
				resp, err := t.tradeStore.GetByMarket(t.ctx, req.MarketID, 0, 1, true)
				if err != nil {
					return nil, err
				}
				trade := &types.Trade{}
				if len(resp) > 0 {
					trade = resp[0]
				}
				return &protoapi.LastTradeResponse{Trade: trade}, nil

			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.LastTradeRequest{},
		PreProcess:   preProcessor,
	}
}
