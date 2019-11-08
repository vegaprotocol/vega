package main

import (
	"io"
	"os"

	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-multierror"
)

const indent string = "  "

// ProcessFiles takes an array of paths to files, reads them in and returns their contents as an array of instruction sets (set per file)
func ProcessFiles(filesWithPath []string) ([]*core.InstructionSet, error) {
	contents, err := openFiles(filesWithPath)
	if err != nil {
		return nil, err
	}

	var errs *multierror.Error
	instructionSets := make([]*core.InstructionSet, len(contents))

	for i, fileContents := range contents {
		instructionSets[i], err = unmarshall(fileContents)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return instructionSets, errs.ErrorOrNil()
}

// Output writes results to the specified file.
func Output(result proto.Message, outputFileWithPath string) error {
	f, err := os.Create(outputFileWithPath)
	if err != nil {
		return err
	}
	return marshall(result, f)
}

func openFiles(filesWithPath []string) ([]*os.File, error) {
	var n = len(filesWithPath)
	readers := make([]*os.File, n)
	var errs *multierror.Error
	var err error

	for i := 0; i < n; i++ {
		readers[i], err = os.Open(filesWithPath[i])
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return readers, errs.ErrorOrNil()
}

func unmarshall(r io.Reader) (*core.InstructionSet, error) {
	var instrSet = &core.InstructionSet{}
	return instrSet, jsonpb.Unmarshal(r, instrSet)
}

func marshall(result proto.Message, out io.Writer) error {
	m := jsonpb.Marshaler{Indent: indent, EmitDefaults: true}
	return m.Marshal(out, result)
}
