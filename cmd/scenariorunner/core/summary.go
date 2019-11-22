package core

import (
	"github.com/golang/protobuf/proto"
)

type summary struct {
	summaryGenerator *SummaryGenerator
}

func NewSummary(summaryGenerator *SummaryGenerator) *summary {
	return &summary{summaryGenerator}
}

func (s *summary) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_MARKET_SUMMARY: s.marketSummary(),
		RequestType_SUMMARY:        s.protocolSummary(),
	}
}

func (s *summary) protocolSummary() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &SummaryRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) { return s.summaryGenerator.Summary(req.GetPagination()) })
	}
	return &PreProcessor{
		MessageShape: &SummaryRequest{},
		PreProcess:   preProcessor,
	}
}

func (s *summary) marketSummary() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &MarketSummaryRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) {
				return s.summaryGenerator.MarketSummary(req.GetMarketID(), req.GetPagination())
			})
	}
	return &PreProcessor{
		MessageShape: &MarketSummaryRequest{},
		PreProcess:   preProcessor,
	}
}
