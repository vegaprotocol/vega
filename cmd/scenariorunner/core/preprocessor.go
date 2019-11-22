package core

import (
	"github.com/golang/protobuf/proto"
)

type PreProcessor struct {
	MessageShape proto.Message // TODO (WG 08/11/2019): This isn't currently used for anything, but I indend to make us of it for generation of help text.
	PreProcess   func(instr *Instruction) (*PreProcessedInstruction, error)
}

type PreProcessorProvider interface {
	PreProcessors() map[RequestType]*PreProcessor
}
