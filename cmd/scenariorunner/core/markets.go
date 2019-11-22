package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

type Markets struct {
	ctx         context.Context
	marketStore *storage.Market
}

func NewMarkets(ctx context.Context, marketStore *storage.Market) *Markets {
	return &Markets{ctx, marketStore}
}

func (m *Markets) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_MARKET_BY_ID: m.marketByID(),
		RequestType_MARKETS:      m.markets(),
	}
}

func (m *Markets) marketByID() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.MarketByIDRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				return m.marketStore.GetByID(req.MarketID)
			})
	}
	return &PreProcessor{
		MessageShape: &protoapi.MarketByIDRequest{},
		PreProcess:   preProcessor,
	}
}

func (m *Markets) markets() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &empty.Empty{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := m.marketStore.GetAll()
				return &protoapi.MarketsResponse{Markets: resp}, err
			})
	}
	return &PreProcessor{
		MessageShape: &empty.Empty{},
		PreProcess:   preProcessor,
	}
}
