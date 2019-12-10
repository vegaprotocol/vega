package node

import (
	"flag"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/fsutil"
)

var (
	Command basecmd.Command

	configPath string
	noChain    bool
	noStores   bool
	withPprof  bool
)

func init() {
	Command.Name = "node"
	Command.Short = "Start a new vega node"

	cmd := flag.NewFlagSet("node", flag.ContinueOnError)
	cmd.StringVar(&configPath, "config-path", fsutil.DefaultVegaDir(), "file path to search for vega config file(s)")
	cmd.BoolVar(&noChain, "no-chain", false, "start the node using the noop chain")
	cmd.BoolVar(&noStores, "no-stores", false, "start the node without stores support")
	cmd.BoolVar(&withPprof, "with-pprof", false, "start the node with pprof support")

	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "%v\n\n", helpNode())
		cmd.PrintDefaults()
	}

	Command.FlagSet = cmd
	Command.Usage = Command.FlagSet.Usage
	Command.Run = runCommand
}

func helpNode() string {
	helpStr := `
Usage: vega node [options]
`
	return strings.TrimSpace(helpStr)
}

func runCommand(args []string) int {
	if err := Command.FlagSet.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(Command.FlagSet.Output(), "%v\n", err)
		return 1
	}

	return 0
}
