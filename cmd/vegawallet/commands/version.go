package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	netv1 "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/version"
	"github.com/spf13/cobra"
)

var (
	versionLong = cli.LongDesc(`
		Get the version of the software and checks if its compatibility with the
		registered networks.

		This is NOT related to the wallet version. To get information about the wallet,
		use the "info" command.
	`)

	versionExample = cli.Examples(`
		# Get the version of the software
		{{.Software}} version
	`)
)

type GetVersionHandler func() (*version.GetVersionResponse, error)

func NewCmdVersion(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*version.GetVersionResponse, error) {
		s, err := netv1.InitialiseStore(paths.New(rf.Home))
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		return version.GetVersionInfo(s, getNetworkVersion)
	}

	return BuildCmdGetVersion(w, h, rf)
}

func BuildCmdGetVersion(w io.Writer, handler GetVersionHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Get the version of the software",
		Long:    versionLong,
		Example: versionExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

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

	if version.IsUnreleased() {
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
	}

	str.Text("Software version:").NextLine().WarningText(resp.Version).NextSection()
	str.Text("Git hash:").NextLine().WarningText(resp.GitHash).NextSection()
	str.Text("Network compatibility:").NextLine()
	nets := make([]string, 0, len(resp.NetworksCompatibility))
	for net := range resp.NetworksCompatibility {
		nets = append(nets, net)
	}
	sort.Strings(nets)
	hasIncompatibilities := false
	for _, net := range nets {
		msg := resp.NetworksCompatibility[net]
		str.Pad().Text("- ").Text(net).Text(": ")
		if msg == "compatible" {
			str.SuccessText(msg)
		} else {
			hasIncompatibilities = true
			str.DangerText(msg)
		}
		str.NextLine()
	}
	str.NextLine()

	if len(resp.NetworksCompatibility) > 0 && hasIncompatibilities {
		str.BlueArrow().InfoText("Note").NextLine()
		str.Text("If you connect to different networks, such as mainnet and testnet, it's normal to have incompatibilities as they may run on different versions, and, may have breaking changes between those versions.").NextLine()
		str.Text("And, because we can't ensure backward or forward compatibility yet, the best option is to download the wallet software for each network, and to switch when you change the network.").NextSection()
	}

	str.RedArrow().DangerText("Important").NextLine()
	str.Text("This command is NOT related to your wallet version.").NextLine()
	str.Bold("This is the version of the software.").NextLine()
	str.Text("To get your wallet version, see the following command:").NextSection()
	str.Code(fmt.Sprintf("%s info --help", os.Args[0])).NextLine()
}
