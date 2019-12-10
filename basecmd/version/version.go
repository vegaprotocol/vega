package version

import (
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/basecmd"
)

var (
	Version     string
	VersionHash string
)

var Command = basecmd.Command{
	Name:  "version",
	Short: "Print the version of the vega node",
	Run: func(args []string) int {
		fmt.Printf("vega version %s (%s)\n", Version, VersionHash)
		return 0
	},
	Usage: func() {
		fmt.Println(helpVersion())
	},
}

func helpVersion() string {
	helpStr := `
usage: vega version

Version prints the vega node version, this is set during the compilation.
`
	return strings.TrimSpace(helpStr)
}
