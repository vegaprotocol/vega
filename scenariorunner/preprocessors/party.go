package preprocessors

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

type Parties struct {
	ctx        context.Context
	partyStore *storage.Party
}

func NewParties(ctx context.Context, partyStore *storage.Party) *Parties {
	return &Parties{ctx, partyStore}
}

func (p *Parties) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_PARTY_BY_ID: p.partyById(),
		core.RequestType_PARTIES:     nil,
	}
}

func (p *Parties) partyById() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.PartyByIDRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := p.partyStore.GetByID(req.PartyID)
				if err != nil {
					return nil, err
				}
				return &protoapi.PartyByIDResponse{Party: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.PartyByIDRequest{},
		PreProcess:   preProcessor,
	}
}

func (p *Parties) parties() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &empty.Empty{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := p.partyStore.GetAll()
				if err != nil {
					return nil, err
				}
				return &protoapi.PartiesResponse{Parties: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &empty.Empty{},
		PreProcess:   preProcessor,
	}
}
