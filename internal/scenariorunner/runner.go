package scenariorunner

import (
	"errors"
)

//ErrNotImplemented throws an error with "NotImplemented" text
var ErrNotImplemented error = errors.New("NotImplemented")

// ProcessInstructions takes a set of instructions and submits them to the protocol
func ProcessInstructions(instructions InstructionSet) (ResultSet, error) {
	return ResultSet{}, ErrNotImplemented
}
