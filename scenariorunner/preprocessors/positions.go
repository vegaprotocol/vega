package preprocessors

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
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

func (p *Positions) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_POSITIONS_BY_PARTY: p.positionsByParty(),
	}
}

func (p *Positions) positionsByParty() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.PositionsByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
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
	return &core.PreProcessor{
		MessageShape: &protoapi.PositionsByPartyRequest{},
		PreProcess:   preProcessor,
	}
}
