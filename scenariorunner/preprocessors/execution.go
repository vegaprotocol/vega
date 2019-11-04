package preprocessors

import (
	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
)

type Execution struct {
	mappings map[string]*core.PreProcessor
}

func NewExecution(e *execution.Engine) *Execution {
	m := map[string]*core.PreProcessor{
		"notifytraderaccount": notifyTraderAccount(e),
		"submitorder":         submitOrder(e),
		"cancelorder":         cancelOrder(e),
		"amendorder":          amendOrder(e),
		"withdraw":            withdraw(e),
	}

	return &Execution{m}
}

func (e *Execution) PreProcessors() map[string]*core.PreProcessor {
	return e.mappings
}

func notifyTraderAccount(e *execution.Engine) *core.PreProcessor {
	req := &types.NotifyTraderAccount{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return nil, e.NotifyTraderAccount(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func submitOrder(e *execution.Engine) *core.PreProcessor {
	req := &types.Order{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(

			func() (proto.Message, error) { return e.SubmitOrder(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func cancelOrder(e *execution.Engine) *core.PreProcessor {
	req := &types.Order{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return e.CancelOrder(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func amendOrder(e *execution.Engine) *core.PreProcessor {
	req := &types.OrderAmendment{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return e.AmendOrder(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func withdraw(e *execution.Engine) *core.PreProcessor {
	req := &types.Withdraw{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return nil, e.Withdraw(req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
