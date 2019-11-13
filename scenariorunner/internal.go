package scenariorunner

import (
	"time"

	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type internalProvider struct {
	vegaTime         *vegatime.Svc
	summaryGenerator *core.SummaryGenerator
}

func newInternalProvider(vegaTime *vegatime.Svc, summaryGenerator *core.SummaryGenerator) *internalProvider {
	return &internalProvider{vegaTime, summaryGenerator}
}

func (p *internalProvider) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_SET_TIME:         p.set(),
		core.RequestType_ADVANCE_TIME:     p.advance(),
		core.RequestType_MARKET_SUMMARY:   p.marketSummary(),
		core.RequestType_PROTOCOL_SUMMARY: p.protocolSummary(),
	}
}

func (p *internalProvider) set() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &core.SetTimeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		time, err := ptypes.Timestamp(req.Time)
		if err != nil {
			return nil, err
		}
		return instr.PreProcess(
			func() (proto.Message, error) { p.SetTime(time); return nil, nil })
	}
	return &core.PreProcessor{
		MessageShape: &core.SetTimeRequest{},
		PreProcess:   preProcessor,
	}
}

func (p *internalProvider) advance() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &core.AdvanceTimeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		duration, err := ptypes.Duration(req.TimeDelta)
		if err != nil {
			return nil, err
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return nil, p.AdvanceTime(duration) })
	}
	return &core.PreProcessor{
		MessageShape: &core.AdvanceTimeRequest{},
		PreProcess:   preProcessor,
	}
}

func (p *internalProvider) protocolSummary() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &core.ProtocolSummaryRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) { return p.summaryGenerator.ProtocolSummary(req.GetPagination()) })
	}
	return &core.PreProcessor{
		MessageShape: &core.ProtocolSummaryRequest{},
		PreProcess:   preProcessor,
	}
}

func (p *internalProvider) marketSummary() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &core.MarketSummaryRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) {
				return p.summaryGenerator.MarketSummary(req.GetMarketID(), req.GetPagination())
			})
	}
	return &core.PreProcessor{
		MessageShape: &core.MarketSummaryRequest{},
		PreProcess:   preProcessor,
	}
}

// SetTime sets protocol time to the provided value
func (p *internalProvider) SetTime(time time.Time) {
	p.vegaTime.SetTimeNow(time)
}

// AdvanceTime advances protocol time by a specified duration
func (p *internalProvider) AdvanceTime(duration time.Duration) error {
	currentTime, err := p.vegaTime.GetTimeNow()
	if err != nil {
		return err
	}
	advancedTime := currentTime.Add(duration)
	p.SetTime(advancedTime)
	return nil
}
