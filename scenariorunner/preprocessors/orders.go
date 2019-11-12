package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
)

type Orders struct {
	ctx        context.Context
	orderStore *storage.Order
	odp        api.OrderDataProvider
}

func NewOrders(ctx context.Context, orderStore *storage.Order, odp api.OrderDataProvider) *Orders {
	return &Orders{ctx, orderStore, odp}
}

func (o *Orders) PreProcessors() map[string]*core.PreProcessor {
	return map[string]*core.PreProcessor{
		"ordersbymarket":     o.ordersByMarket(),
		"ordersbyparty":      o.ordersByParty(),
		"orderbymarketandid": o.orderByMarketAndID(),
		"orderbyreference":   o.orderByReference(),
	}
}

func (o *Orders) ordersByMarket() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrdersByMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrdersByMarket(o.ctx, req, o.odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrdersByMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) ordersByParty() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrdersByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrdersByParty(o.ctx, req, o.odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrdersByPartyRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) orderByMarketAndID() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrderByMarketAndIdRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrderByMarketAndId(o.ctx, req, o.odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrderByMarketAndIdRequest{},
		PreProcess:   preProcessor,
	}
}

func (o *Orders) orderByReference() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrderByReferenceRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrderByReference(o.ctx, req, o.odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrderByReferenceRequest{},
		PreProcess:   preProcessor,
	}
}
