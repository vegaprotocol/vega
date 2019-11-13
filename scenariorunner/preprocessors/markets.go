package preprocessors

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
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

func (m *Markets) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_MARKET_BY_ID: m.marketByID(),
		core.RequestType_MARKETS:      m.markets(),
	}
}

func (m *Markets) marketByID() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.MarketByIDRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				m.commitStore()
				return m.marketStore.GetByID(req.MarketID)
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.MarketByIDRequest{},
		PreProcess:   preProcessor,
	}
}

func (m *Markets) markets() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &empty.Empty{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				m.commitStore()
				resp, err := m.marketStore.GetAll()
				return &protoapi.MarketsResponse{Markets: resp}, err
			})
	}
	return &core.PreProcessor{
		MessageShape: &empty.Empty{},
		PreProcess:   preProcessor,
	}
}

func (m *Markets) commitStore() {
	m.marketStore.Commit()
}
