package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
)

type Orders struct {
	ctx        context.Context
	orderStore *storage.Order
}

func NewOrders(ctx context.Context, orderStore *storage.Order) *Orders {
	return &Orders{ctx, orderStore}
}

func (o *Orders) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_ORDERS_BY_MARKET:       o.ordersByMarket(),
		RequestType_ORDERS_BY_PARTY:        o.ordersByParty(),
		RequestType_ORDER_BY_MARKET_AND_ID: o.orderByMarketAndID(),
		RequestType_ORDER_BY_REFERENCE:     o.orderByReference(),
		RequestType_MARKET_DEPTH:           o.marketDepth(),
	}
}

func (o *Orders) ordersByMarket() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.OrdersByMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		pagination := GetDefaultPagination(req.Pagination)
		return instr.PreProcess(
			func() (proto.Message, error) {
				//o.commitStore()
				resp, err := o.orderStore.GetByMarket(o.ctx, req.MarketID, pagination.Skip, pagination.Limit, pagination.Descending, &req.Open)
				if err != nil {
					return nil, err
				}
				return &protoapi.OrdersByMarketResponse{Orders: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.OrdersByMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) ordersByParty() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.OrdersByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		pagination := GetDefaultPagination(req.Pagination)
		return instr.PreProcess(
			func() (proto.Message, error) {
				//o.commitStore()
				resp, err := o.orderStore.GetByParty(o.ctx, req.PartyID, pagination.Skip, pagination.Limit, pagination.Descending, &req.Open)
				if err != nil {
					return nil, err
				}
				return &protoapi.OrdersByPartyResponse{Orders: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.OrdersByPartyRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) orderByMarketAndID() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.OrderByMarketAndIdRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				//o.commitStore()
				resp, err := o.orderStore.GetByMarketAndID(o.ctx, req.MarketID, req.OrderID)
				if err != nil {
					return nil, err
				}
				return &protoapi.OrderByMarketAndIdResponse{Order: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.OrderByMarketAndIdRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) orderByReference() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.OrderByReferenceRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				//o.commitStore()
				resp, err := o.orderStore.GetByReference(o.ctx, req.Reference)
				if err != nil {
					return nil, err
				}
				return &protoapi.OrderByMarketAndIdResponse{Order: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.OrderByReferenceRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) marketDepth() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.MarketDepthRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				//o.commitStore()
				resp, err := o.orderStore.GetMarketDepth(o.ctx, req.MarketID)
				return &protoapi.MarketDepthResponse{MarketID: resp.MarketID, Buy: resp.Buy, Sell: resp.Sell}, err
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.MarketDepthRequest{},
		PreProcess:   preProcessor,
	}
}
