package core

import (
	"github.com/golang/protobuf/proto"
)

type PreProcessor struct {
	MessageShape proto.Message
	PreProcess   func(instr *Instruction) (*PreProcessedInstruction, error)
}

type PreProcessorProvider interface {
	PreProcessors() map[string]*PreProcessor
}
