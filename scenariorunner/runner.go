package scenariorunner

import (
	"errors"
	"strings"

	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/scenariorunner/preprocessors"

	"github.com/hashicorp/go-multierror"
)

var (
	ErrNotImplemented error = errors.New("Not implemented")
)

type ScenarioRunner struct {
	providers []core.PreProcessorProvider
}

// NewScenarioRunner returns a pointer to new instance of scenario runner
func NewScenarioRunner() (*ScenarioRunner, error) {

	executionPreprocessor, err := preprocessors.NewExecution()
	if err != nil {
		return nil, err
	}
	return &ScenarioRunner{
		providers: []core.PreProcessorProvider{
			executionPreprocessor,
		},
	}, nil
}

// ProcessInstructions takes a set of instructions and submits them to the protocol
func (sr ScenarioRunner) ProcessInstructions(instrSet core.InstructionSet) (core.ResultSet, error) {
	var processed, omitted uint64
	n := len(instrSet.Instructions)
	results := make([]*core.InstructionResult, n)
	var errors *multierror.Error

	preProcessors := sr.providers[0].PreProcessors()

	for i, instr := range instrSet.Instructions {
		// TODO (WG 01/11/2019) matching by lower case by convention only, enforce with a custom type
		preProcessor, ok := preProcessors[strings.ToLower(instr.Request)]
		if !ok {
			errors = multierror.Append(errors, core.ErrInstructionNotSupported)
			omitted++
			continue
		}
		p, err := preProcessor.PreProcess(instr)
		if err != nil {
			errors = multierror.Append(errors, err)
			omitted++
			continue
		}
		res, err := p.Result()
		if err != nil {
			errors = multierror.Append(errors, err)
			omitted++
			continue
		}
		results[i] = res
		processed++
	}

	md := &core.Metadata{
		InstructionsProcessed: processed,
		InstructionsOmitted:   omitted,
	}

	return core.ResultSet{
		Summary: md,
		Results: results,
	}, errors.ErrorOrNil()
}
