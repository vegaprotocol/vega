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
	orderStore *storage.Order
	mappings   map[string]*core.PreProcessor
}

func NewOrders(ctx context.Context, orderStore *storage.Order, odp api.OrderDataProvider) *Orders {
	m := map[string]*core.PreProcessor{
		"ordersbymarket":     ordersByMarket(ctx, odp),
		"ordersbyparty":      ordersByParty(ctx, odp),
		"orderbymarketandid": orderByMarketAndID(ctx, odp),
		"orderbyreference":   orderByReference(ctx, odp),
	}
	return &Orders{orderStore, m}
}

func (o *Orders) PreProcessors() map[string]*core.PreProcessor {
	return o.mappings
}

func ordersByMarket(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrdersByMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrdersByMarket(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrdersByMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func ordersByParty(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrdersByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrdersByParty(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrdersByPartyRequest{},
		PreProcess:   preProcessor,
	}
}

func orderByMarketAndID(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrderByMarketAndIdRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrderByMarketAndId(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrderByMarketAndIdRequest{},
		PreProcess:   preProcessor,
	}
}

func orderByReference(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.OrderByReferenceRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrderByReference(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.OrderByReferenceRequest{},
		PreProcess:   preProcessor,
	}
}
