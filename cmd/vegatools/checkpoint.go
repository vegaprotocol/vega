package tools

import (
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/vegatools/checkpoint"
)

type checkpointCmd struct {
	config.OutputFlag

	InPath   string `short:"f" long:"file" required:"true" description:"input file to parse"`
	OutPath  string `short:"o" long:"out" description:"output file to write to [default is STDOUT]"`
	Validate bool   `short:"v" long:"validate" description:"validate contents of the checkpoint file"`
	Generate bool   `short:"g" long:"generate" description:"The chain to be imported"`
	Dummy    bool   `short:"d" long:"dummy" description:"generate a dummy file [added for debugging, but could be useful]"`
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
