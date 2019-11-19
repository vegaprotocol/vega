package preprocessors

import (
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
)

type Summary struct {
	summaryGenerator *core.SummaryGenerator
}

func NewSummary(summaryGenerator *core.SummaryGenerator) *Summary {
	return &Summary{summaryGenerator}
}

func (s *Summary) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_MARKET_SUMMARY: s.marketSummary(),
		core.RequestType_SUMMARY:        s.protocolSummary(),
	}
}

func (s *Summary) protocolSummary() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &core.SummaryRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) { return s.summaryGenerator.Summary(req.GetPagination()) })
	}
	return &core.PreProcessor{
		MessageShape: &core.SummaryRequest{},
		PreProcess:   preProcessor,
	}
}

func (s *Summary) marketSummary() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &core.MarketSummaryRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) {
				return s.summaryGenerator.MarketSummary(req.GetMarketID(), req.GetPagination())
			})
	}
	return &core.PreProcessor{
		MessageShape: &core.MarketSummaryRequest{},
		PreProcess:   preProcessor,
	}
}
