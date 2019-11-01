package scenariorunner

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
)

// NewInstruction returns a new instruction from the request and proto message.
func NewInstruction(request string, message proto.Message) (*Instruction, error) {
	any, err := marshalAny(message)
	if err != nil {
		return nil, err
	}
	return &Instruction{
		Request: request,
		Message: any,
	}, nil
}

// NewResult wraps a response and an error in InstructionResult.
func (instr Instruction) NewResult(response proto.Message, err error) (*InstructionResult, error) {
	errText := ""
	if err != nil {
		errText = err.Error()
	}

	//TODO (WG 01/11/2019): A bit of a hack, but probably better than reflection or additional code returning typless nil when processing instructions.
	//Still, there might be a better way around it, I just don't know it yet.
	if response == nil || response.String() == "<nil>" {
		response = &empty.Empty{}
	}

	any, err := marshalAny(response)
	if err != nil {
		return nil, err
	}

	return &InstructionResult{
		Response:    any,
		Error:       errText,
		Instruction: &instr,
	}, nil
}

func marshalAny(pb proto.Message) (*any.Any, error) {
	value, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return &any.Any{TypeUrl: proto.MessageName(pb), Value: value}, nil
}
