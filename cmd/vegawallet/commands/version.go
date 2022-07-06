package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/version"

	"github.com/spf13/cobra"
)

var (
	versionLong = cli.LongDesc(`
		Get the version of the software.

		This is NOT related to the wallet version. To get information about the wallet,
		use the "info" command.
	`)

	versionExample = cli.Examples(`
		# Get the version of the software
		{{.Software}} version
	`)
)

type GetVersionHandler func() *version.GetVersionResponse

func NewCmdVersion(w io.Writer, rf *RootFlags) *cobra.Command {
	return BuildCmdGetVersion(w, version.GetVersionInfo, rf)
}

func BuildCmdGetVersion(w io.Writer, handler GetVersionHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Get the version of the software",
		Long:    versionLong,
		Example: versionExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp := handler()

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintGetVersionResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintGetVersionResponse(w io.Writer, resp *version.GetVersionResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Software version:").NextLine().WarningText(resp.Version).NextSection()
	str.Text("Git hash:").NextLine().WarningText(resp.GitHash).NextSection()

	str.RedArrow().DangerText("Important").NextLine()
	str.Text("This command is NOT related to your wallet version.").NextLine()
	str.Bold("This is the version of the software.").NextLine()
	str.Text("To get your wallet version, see the following command:").NextSection()
	str.Code(fmt.Sprintf("%s info --help", os.Args[0])).NextLine()
}
