package preprocessors

import (
	"time"

	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type Time struct {
	vegaTime *vegatime.Svc
}

func NewTime(vegaTime *vegatime.Svc) *Time {
	return &Time{vegaTime}
}

func (t *Time) PreProcessors() map[string]*core.PreProcessor {
	return map[string]*core.PreProcessor{
		"settime":     t.set(),
		"advancetime": t.advance(),
	}
}

func (t *Time) set() *core.PreProcessor {
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
			func() (proto.Message, error) { t.SetTime(time); return nil, nil })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func (t *Time) advance() *core.PreProcessor {
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
			func() (proto.Message, error) { return nil, t.AdvanceTime(duration) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// SetTime sets protocol time to the provided value
func (t *Time) SetTime(time time.Time) {
	t.vegaTime.SetTimeNow(time)
}

// AdvanceTime advances protocol time by a specified duration
func (t *Time) AdvanceTime(duration time.Duration) error {
	currentTime, err := t.vegaTime.GetTimeNow()
	if err != nil {
		return err
	}
	advancedTime := currentTime.Add(duration)
	t.SetTime(advancedTime)
	return nil
}
