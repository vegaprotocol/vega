package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
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

func (p *Parties) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_PARTY_BY_ID: p.partyByID(),
		RequestType_PARTIES:     nil,
	}
}

func (p *Parties) partyByID() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.PartyByIDRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
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
	return &PreProcessor{
		MessageShape: &protoapi.PartyByIDRequest{},
		PreProcess:   preProcessor,
	}
}

func (p *Parties) parties() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &empty.Empty{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
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
	return &PreProcessor{
		MessageShape: &empty.Empty{},
		PreProcess:   preProcessor,
	}
}
