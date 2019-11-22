package core

import (
	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/golang/protobuf/proto"
)

type Execution struct {
	engine *execution.Engine
}

func NewExecution(e *execution.Engine) *Execution {
	return &Execution{e}
}

func (e *Execution) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_NOTIFY_TRADER_ACCOUNT: e.notifyTraderAccount(),
		RequestType_SUBMIT_ORDER:          e.submitOrder(),
		RequestType_CANCEL_ORDER:          e.cancelOrder(),
		RequestType_AMEND_ORDER:           e.amendOrder(),
		RequestType_WITHDRAW:              e.withdraw(),
	}
}

func (e *Execution) notifyTraderAccount() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.NotifyTraderAccountRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				err := e.engine.NotifyTraderAccount(req.Notif)
				if err != nil {
					return nil, err
				}
				return &protoapi.NotifyTraderAccountResponse{Submitted: true}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.NotifyTraderAccountRequest{},
		PreProcess:   preProcessor,
	}
}

func (e *Execution) submitOrder() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.SubmitOrderRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				return e.engine.SubmitOrder(getOrderFromSubmission(req.Submission))
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.SubmitOrderRequest{},
		PreProcess:   preProcessor,
	}
}

func (e *Execution) cancelOrder() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.CancelOrderRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return e.engine.CancelOrder(getOrderFromCancellation(req.Cancellation)) })
	}
	return &PreProcessor{
		MessageShape: &types.Order{},
		PreProcess:   preProcessor,
	}
}

func (e *Execution) amendOrder() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.AmendOrderRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return e.engine.AmendOrder(req.Amendment) })
	}
	return &PreProcessor{
		MessageShape: &types.OrderAmendment{},
		PreProcess:   preProcessor,
	}
}

func (e *Execution) withdraw() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.WithdrawRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				err := e.engine.Withdraw(req.Withdraw)
				if err != nil {
					return nil, err
				}
				return &protoapi.WithdrawResponse{Success: true}, nil
			},
		)
	}
	return &PreProcessor{
		MessageShape: &protoapi.WithdrawRequest{},
		PreProcess:   preProcessor,
	}
}

func getOrderFromSubmission(sub *types.OrderSubmission) *types.Order {
	order := &types.Order{
		Id:          sub.Id,
		MarketID:    sub.MarketID,
		PartyID:     sub.PartyID,
		Side:        sub.Side,
		Price:       sub.Price,
		Size:        sub.Size,
		Remaining:   sub.Size,
		TimeInForce: sub.TimeInForce,
		Type:        sub.Type,
		ExpiresAt:   sub.ExpiresAt}
	return order
}

func getOrderFromCancellation(sub *types.OrderCancellation) *types.Order {
	order := &types.Order{
		Id:       sub.OrderID,
		MarketID: sub.MarketID,
		PartyID:  sub.PartyID,
	}
	return order
}
