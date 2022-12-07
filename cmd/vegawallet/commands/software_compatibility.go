package cmd

import (
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	netv1 "code.vegaprotocol.io/vega/wallet/network/store/v1"
	wversion "code.vegaprotocol.io/vega/wallet/version"
	"github.com/spf13/cobra"
)

var (
	softwareCompatibilityLong = `Check the compatibility between this software and all the registered networks.

# What are these incompatibilities?

Breaking changes may be introduced between software versions deployed on the networks.
And, because the wallet software is deeply tied to the network APIs, when it is run 
against a different version of the network, some requests may fail.

Currently there's no guarantee of backward or forward compatibility, but that will 
change when the network is officially defined as stable.

# My software is said to be incompatible, what can I do when m?

The best option is to:

1. Download the version of the wallet software matching the version running on the network at:
   https://github.com/vegaprotocol/vega/releases
   Example: If the network is running 0.57.1, download the wallet software with the version 0.57.1.

2. Then, switch to the wallet software matching the network version.

# Will I have to do that forever?

No. This will not be a problem once the network is officially defined as stable.
`

	softwareCompatibilityExample = cli.Examples(`
		# Check the software compatibility against all registered networks
		{{.Software}} software compatibility
	`)
)

type CheckSoftwareCompatibilityHandler func() (*wversion.CheckSoftwareCompatibilityResponse, error)

func NewCmdSoftwareCompatibility(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func() (*wversion.CheckSoftwareCompatibilityResponse, error) {
		s, err := netv1.InitialiseStore(paths.New(rf.Home))
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		return wversion.CheckSoftwareCompatibility(s, wversion.GetNetworkVersionThroughGRPC)
	}

	return BuildCmdCheckSoftwareCompatibility(w, h, rf)
}

func BuildCmdCheckSoftwareCompatibility(w io.Writer, handler CheckSoftwareCompatibilityHandler, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "compatibility",
		Short:   "Check the compatibility between this software and all the registered networks",
		Long:    softwareCompatibilityLong,
		Example: softwareCompatibilityExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := handler()
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintCheckSoftwareIncompatibilityResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	return cmd
}

func PrintCheckSoftwareIncompatibilityResponse(w io.Writer, resp *wversion.CheckSoftwareCompatibilityResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

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
}
