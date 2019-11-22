package core

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type Time struct {
	time *TimeControl
}

func NewTime(time *TimeControl) *Time {
	return &Time{time}
}

func (t *Time) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_SET_TIME:     t.set(),
		RequestType_ADVANCE_TIME: t.advance(),
	}
}

func (t *Time) set() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &SetTimeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		time, err := ptypes.Timestamp(req.Time)
		if err != nil {
			return nil, err
		}
		return instr.PreProcess(
			func() (proto.Message, error) { t.time.SetTime(time); return nil, nil })
	}
	return &PreProcessor{
		MessageShape: &SetTimeRequest{},
		PreProcess:   preProcessor,
	}
}

func (t *Time) advance() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &AdvanceTimeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
		}
		duration, err := ptypes.Duration(req.TimeDelta)
		if err != nil {
			return nil, err
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return nil, t.time.AdvanceTime(duration) })
	}
	return &PreProcessor{
		MessageShape: &AdvanceTimeRequest{},
		PreProcess:   preProcessor,
	}
}
