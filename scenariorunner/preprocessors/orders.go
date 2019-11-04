package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
)

type Orders struct {
	mappings map[string]*core.PreProcessor
}

func NewOrders(ctx context.Context, odp api.OrderDataProvider) *Orders {
	m := map[string]*core.PreProcessor{
		"ordersbymarket":     ordersByMarket(ctx, odp),
		"ordersbyparty":      ordersByParty(ctx, odp),
		"orderbymarketandid": orderByMarketAndID(ctx, odp),
		"orderbyreference":   orderByReference(ctx, odp),
	}
	return &Orders{m}
}

func (o *Orders) PreProcessors() map[string]*core.PreProcessor {
	return o.mappings
}

func ordersByMarket(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	req := &protoapi.OrdersByMarketRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrdersByMarket(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func ordersByParty(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	req := &protoapi.OrdersByPartyRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrdersByParty(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func orderByMarketAndID(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	req := &protoapi.OrderByMarketAndIdRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrderByMarketAndId(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func orderByReference(ctx context.Context, odp api.OrderDataProvider) *core.PreProcessor {
	req := &protoapi.OrderByReferenceRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return api.ProcessOrderByReference(ctx, req, odp) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
