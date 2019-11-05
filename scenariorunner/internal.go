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

func (p *internalProvider) PreProcessors() map[string]*core.PreProcessor {
	return map[string]*core.PreProcessor{
		"settime":         p.set(),
		"advancetime":     p.advance(),
		"protocolsummary": p.marketSummary(),
		"marketsummary":   p.protocolSummary(),
	}
}

func (p *internalProvider) set() *core.PreProcessor {
	req := &core.SetTimeRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
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
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func (p *internalProvider) advance() *core.PreProcessor {
	req := &core.AdvanceTimeRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
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
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func (p *internalProvider) protocolSummary() *core.PreProcessor {
	req := &core.ProtocolSummaryRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) { return p.summaryGenerator.ProtocolSummary(req.GetPagination()) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func (p *internalProvider) marketSummary() *core.PreProcessor {
	req := &core.MarketSummaryRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}

		return instr.PreProcess(
			func() (proto.Message, error) {
				return p.summaryGenerator.MarketSummary(req.GetMarketID(), req.GetPagination())
			})
	}
	return &core.PreProcessor{
		MessageShape: req,
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
