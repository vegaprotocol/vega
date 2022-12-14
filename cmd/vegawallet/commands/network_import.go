package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	netstore "code.vegaprotocol.io/vega/wallet/network/store/v1"

	"github.com/spf13/cobra"
)

var (
	importNetworkLong = cli.LongDesc(`
		Import a network configuration from a file or an URL.
	`)

	importNetworkExample = cli.Examples(`
		# import a network configuration from a file
		{{.Software}} network import --from-file PATH_TO_NETWORK

		# import a network configuration from an URL
		{{.Software}} network import --from-url URL_TO_NETWORK

		# overwrite existing network configuration
		{{.Software}} network import --from-url URL_TO_NETWORK --force

		# import a network configuration with a different name
		{{.Software}} network import --from-url URL_TO_NETWORK --with-name NEW_NAME
	`)
)

type ImportNetworkFromSourceHandler func(api.AdminImportNetworkParams) (api.AdminImportNetworkResult, error)

func NewCmdImportNetwork(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminImportNetworkParams) (api.AdminImportNetworkResult, error) {
		vegaPaths := paths.New(rf.Home)

		s, err := netstore.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminImportNetworkResult{}, fmt.Errorf("couldn't initialise networks store: %w", err)
		}
		importNetwork := api.NewAdminImportNetwork(s)
		rawResult, errorDetails := importNetwork.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errorDetails != nil {
			return api.AdminImportNetworkResult{}, errors.New(errorDetails.Data)
		}

		return rawResult.(api.AdminImportNetworkResult), nil
	}

	return BuildCmdImportNetwork(w, h, rf)
}

func BuildCmdImportNetwork(w io.Writer, handler ImportNetworkFromSourceHandler, rf *RootFlags) *cobra.Command {
	f := &ImportNetworkFlags{}

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Import a network configuration",
		Long:    importNetworkLong,
		Example: importNetworkExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := f.Validate()
			if err != nil {
				return err
			}

			resp, err := handler(req)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintImportNetworkResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&f.FilePath,
		"from-file",
		"",
		"Path to the file containing the network configuration to import",
	)
	cmd.Flags().StringVar(&f.URL,
		"from-url",
		"",
		"URL of the file containing the network configuration to import",
	)
	cmd.Flags().StringVar(&f.Name,
		"with-name",
		"",
		"Change the name of the imported network",
	)
	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Overwrite the existing network if it has the same name",
	)

	return cmd
}

type ImportNetworkFlags struct {
	FilePath string
	URL      string
	Name     string
	Force    bool
}

func (f *ImportNetworkFlags) Validate() (api.AdminImportNetworkParams, error) {
	if len(f.FilePath) == 0 && len(f.URL) == 0 {
		return api.AdminImportNetworkParams{}, flags.OneOfFlagsMustBeSpecifiedError("from-file", "from-url")
	}

	if len(f.FilePath) != 0 && len(f.URL) != 0 {
		return api.AdminImportNetworkParams{}, flags.MutuallyExclusiveError("from-file", "from-url")
	}

	return api.AdminImportNetworkParams{
		FilePath:  f.FilePath,
		URL:       f.URL,
		Name:      f.Name,
		Overwrite: f.Force,
	}, nil
}

func PrintImportNetworkResponse(w io.Writer, resp api.AdminImportNetworkResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Importing the network succeeded").NextSection()
	str.Text("Name:").NextLine().WarningText(resp.Name).NextLine()
	str.Text("File path:").NextLine().WarningText(resp.FilePath).NextLine()
}
