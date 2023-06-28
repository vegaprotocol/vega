package tools

import (
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/vegatools/checkpoint"
)

type checkpointCmd struct {
	config.OutputFlag

	InPath   string `description:"input file to parse"                                              long:"file"     required:"true" short:"f"`
	OutPath  string `description:"output file to write to [default is STDOUT]"                      long:"out"      short:"o"`
	Validate bool   `description:"validate contents of the checkpoint file"                         long:"validate" short:"v"`
	Generate bool   `description:"The chain to be imported"                                         long:"generate" short:"g"`
	Dummy    bool   `description:"generate a dummy file [added for debugging, but could be useful]" long:"dummy"    short:"d"`
}

func (opts *checkpointCmd) Execute(_ []string) error {
	checkpoint.Run(
		opts.InPath,
		opts.OutPath,
		opts.Generate,
		opts.Validate,
		opts.Dummy,
	)
	return nil
}
