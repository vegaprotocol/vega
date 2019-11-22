package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/trades"

	"github.com/golang/protobuf/proto"
)

type Positions struct {
	ctx          context.Context
	tradeService *trades.Svc
}

func NewPositions(ctx context.Context, tradeService *trades.Svc) *Positions {
	return &Positions{ctx, tradeService}
}

func (p *Positions) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_POSITIONS_BY_PARTY: p.positionsByParty(),
	}
}

func (p *Positions) positionsByParty() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.PositionsByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := p.tradeService.GetPositionsByParty(p.ctx, req.PartyID)
				if err != nil {
					return nil, err
				}
				return &protoapi.PositionsByPartyResponse{Positions: resp}, nil
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.PositionsByPartyRequest{},
		PreProcess:   preProcessor,
	}
}
