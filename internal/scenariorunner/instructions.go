package scenariorunner

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"

	proto "github.com/golang/protobuf/proto"
)

// Instruction holds together instruction type, corresponding protobuf message and an optional description
type Instruction struct {
	Type        string
	Message     proto.Message
	Description string
}

// InstructionSet contains a set of instructions along with an optional description
type InstructionSet struct {
	Instructions []Instruction
	Descrption   string
}

// InstructionResult holds together pointer to an instruction along with the corresponding response or error if instruction could not be processed
type InstructionResult struct {
	Instruction *Instruction
	Response    proto.Message
	Error       error
}

// Metadata holds basic details about a result of processing a given InstructionSet
type Metadata struct {
	InstructionsProcessed int
	InstructionsOmitted   int
	TradersGenerated      int
	ProcessingTime        time.Duration
	FinalOrderBooks       []types.MarketDepth
}

// ResultSet contains a set InstructionResults corresponding to a given InstructionSet along with the associated metadata
type ResultSet struct {
	Results []InstructionResult
	Summary Metadata
	//TODO (WG 16/10/19): Once config is added include it in the result set
	//TODO (WG 16/10/19): Add version of `trading-core`
}
