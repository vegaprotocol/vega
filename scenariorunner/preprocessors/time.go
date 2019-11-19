package preprocessors

import (
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type Time struct {
	time *core.TimeControl
}

func NewTime(time *core.TimeControl) *Time {
	return &Time{time}
}

func (t *Time) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_SET_TIME:     t.set(),
		core.RequestType_ADVANCE_TIME: t.advance(),
	}
}

func (t *Time) set() *core.PreProcessor {
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
			func() (proto.Message, error) { t.time.SetTime(time); return nil, nil })
	}
	return &core.PreProcessor{
		MessageShape: &core.SetTimeRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Time) advance() *core.PreProcessor {
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
			func() (proto.Message, error) { return nil, t.time.AdvanceTime(duration) })
	}
	return &core.PreProcessor{
		MessageShape: &core.AdvanceTimeRequest{},
		PreProcess:   preProcessor,
	}
}
