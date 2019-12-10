package version

import (
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/logging"
)

var Command = basecmd.Command{
	Name:  "version",
	Short: "Print the version of the vega node",
	Run: func(_ *logging.Logger, args []string) int {
		fmt.Printf("vega version %s (%s)\n", basecmd.Version, basecmd.VersionHash)
		return 0
	},
	Usage: func() {
		fmt.Println(helpVersion())
	},
}

func helpVersion() string {
	helpStr := `
Usage: vega version

Version prints the vega node version, this is set during the compilation.
`
	return strings.TrimSpace(helpStr)
}
