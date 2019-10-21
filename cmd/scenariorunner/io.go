package main

import (
	"io"
	"os"

	sr "code.vegaprotocol.io/vega/internal/scenariorunner"

	"github.com/golang/protobuf/jsonpb"
	"github.com/hashicorp/go-multierror"
)

// ProcessFiles takes an array of paths to files, reads them in and returns their contents as an array of instruction sets (set per file)
func ProcessFiles(filesWithPath []string) ([]*sr.InstructionSet, error) {
	contents, err := readFiles(filesWithPath)
	if err != nil {
		return nil, err
	}

	var errors *multierror.Error
	instructionSets := make([]*sr.InstructionSet, len(contents))

	for i, fileContents := range contents {
		instructionSets[i], err = unmarshall(fileContents)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return instructionSets, errors.ErrorOrNil()
}

func readFiles(filesWithPath []string) ([]*os.File, error) {
	var n = len(filesWithPath)
	readers := make([]*os.File, n)
	var errors *multierror.Error
	var err error

	for i := 0; i < n; i++ {
		readers[i], err = os.Open(filesWithPath[i])
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return readers, errors.ErrorOrNil()
}

func unmarshall(r io.Reader) (*sr.InstructionSet, error) {
	var instrSet = &sr.InstructionSet{}
	err := jsonpb.Unmarshal(r, instrSet)
	if err != nil {
		return nil, err
	}
	return instrSet, nil
}

func marshall(result *sr.ResultSet, out io.Writer) error {
	m := jsonpb.Marshaler{Indent: "  "}
	return m.Marshal(out, result)
}
