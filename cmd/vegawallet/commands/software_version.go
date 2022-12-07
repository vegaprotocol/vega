package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	netv1 "code.vegaprotocol.io/vega/wallet/network/store/v1"
	wversion "code.vegaprotocol.io/vega/wallet/version"
	"github.com/spf13/cobra"
)

var (
	softwareVersionLong = cli.LongDesc(`
		Get the version of the software and checks if its compatibility with the
		registered networks.

		This is NOT related to the wallet version. To get information about the wallet,
		use the "info" command.
	`)

	softwareVersionExample = cli.Examples(`
		# Get the version of the software
		{{.Software}} software version
	`)
)

type GetSoftwareVersionHandler func() (*wversion.GetSoftwareVersionResponse, error)

func NewCmdSoftwareVersion(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*wversion.GetSoftwareVersionResponse, error) {
		s, err := netv1.InitialiseStore(paths.New(rf.Home))
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		return wversion.GetVersionInfo(s, wversion.GetNetworkVersionThroughGRPC), nil
	}

	return BuildCmdGetVersion(w, h, rf)
}

func BuildCmdGetVersion(w io.Writer, handler GetSoftwareVersionHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Get the version of the software",
		Long:    softwareVersionLong,
		Example: softwareVersionExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintGetSoftwareVersionResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintGetSoftwareVersionResponse(w io.Writer, resp *wversion.GetSoftwareVersionResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if wversion.IsUnreleased() {
		str.CrossMark().DangerText("You are running an unreleased version of the software (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
	}

	str.Text("Software version:").NextLine().WarningText(resp.Version).NextSection()
	str.Text("Git hash:").NextLine().WarningText(resp.GitHash).NextSection()
	str.Text("Network compatibility:").NextLine()

	hasIncompatibilities := false

	for _, networkCompatibility := range resp.NetworksCompatibility {
		str.Text("- ").Bold(networkCompatibility.Network).Text(":").NextLine()
		if networkCompatibility.Error != nil {
			str.Pad().WarningText("Unable to determine this network version:").NextLine()
			str.Pad().WarningText(networkCompatibility.Error.Error())
		} else {
			str.Pad().Text("This network is running the version ").InfoText(networkCompatibility.RetrievedVersion).NextLine()
			if !networkCompatibility.IsCompatible {
				hasIncompatibilities = true
				str.Pad().DangerBold("Incompatible.")
			} else {
				str.Pad().SuccessBold("Compatible.")
			}
		}
		str.NextLine()
	}
	str.NextLine()

	if len(resp.NetworksCompatibility) > 0 && hasIncompatibilities {
		str.BlueArrow().InfoText("What are these incompatibilities?").NextLine()
		str.Text("Breaking changes may be introduced between software versions deployed on the networks. And, because the wallet software is deeply tied to the network APIs, when it is run against a different version of the network, some requests may fail.").NextLine()
		str.Text("Currently there's no guarantee of backward or forward compatibility, but that will change when the network is officially defined as stable.").NextSection()

		str.BlueArrow().InfoText("What can I do then?").NextLine()
		str.Text("The best option is to:").NextLine()
		str.Text("1. Download the version of the wallet software matching the version running on the network at:").NextLine()
		str.Text("   ").Underline("https://github.com/vegaprotocol/vega/releases").NextLine()
		str.Text("   Example: If the network is running 0.57.1, download the wallet software with the version 0.57.1.").NextLine()
		str.Text("2. Then, switch to the wallet software matching the network version.").NextSection()

		str.BlueArrow().InfoText("Will I have to do that forever?").NextLine()
		str.Text("No. This will not be a problem once the network is officially defined as stable.").NextSection()
	}

	str.RedArrow().DangerText("Important").NextLine()
	str.Text("This command is NOT related to your wallet version.").NextLine()
	str.Bold("This is the version of the software.").NextLine()
	str.Text("To get your wallet version, see the following command:").NextSection()
	str.Code(fmt.Sprintf("%s info --help", os.Args[0])).NextLine()
}
