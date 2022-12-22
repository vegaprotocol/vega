package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"github.com/spf13/cobra"
)

var rootExamples = cli.Examples(`
	# Specify a custom Vega home directory
	{{.Software}} --home PATH_TO_DIR COMMAND

	# Change the output to JSON
	{{.Software}} --output json COMMAND

	# Disable colors on output using environment variable
	NO_COLOR=1 {{.Software}} COMMAND
`)

func NewCmdRoot(w io.Writer) *cobra.Command {
	return BuildCmdRoot(w)
}

func BuildCmdRoot(w io.Writer) *cobra.Command {
	f := &RootFlags{}

	cmd := &cobra.Command{
		Use:           os.Args[0],
		Short:         "The Vega wallet",
		Long:          "The Vega wallet",
		Example:       rootExamples,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// The `__complete` command is being run to build up the auto-completion
			// file. We should skip any verification to not temper with the process.
			// Any additional printing will end up in the auto-completion registry.
			// The `completion` command output the completion script for a given
			// shell, that should not be tempered with. We should skip it as well.
			if cmd.Name() == "__complete" || cmd.Name() == "completion" {
				return nil
			}

			if err := f.Validate(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&f.Output,
		"output", "o",
		flags.InteractiveOutput,
		fmt.Sprintf("Specify the output format: %v", flags.AvailableOutputs),
	)
	cmd.PersistentFlags().StringVar(&f.Home,
		"home",
		"",
		"Specify the location of a custom Vega home",
	)

	_ = cmd.MarkPersistentFlagDirname("home")
	_ = cmd.RegisterFlagCompletionFunc("output", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return flags.AvailableOutputs, cobra.ShellCompDirectiveDefault
	})

	// Root commands
	cmd.AddCommand(NewCmdInit(w, f))

	// Sub-commands
	cmd.AddCommand(NewCmdAPIToken(w, f))
	cmd.AddCommand(NewCmdKey(w, f))
	cmd.AddCommand(NewCmdMessage(w, f))
	cmd.AddCommand(NewCmdNetwork(w, f))
	cmd.AddCommand(NewCmdPassphrase(w, f))
	cmd.AddCommand(NewCmdPermissions(w, f))
	cmd.AddCommand(NewCmdRawTransaction(w, f))
	cmd.AddCommand(NewCmdService(w, f))
	cmd.AddCommand(NewCmdShell(w, f))
	cmd.AddCommand(NewCmdSoftware(w, f))
	cmd.AddCommand(NewCmdTransaction(w, f))

	// Wallet commands
	// We don't have a wrapper sub-command for wallet commands.
	cmd.AddCommand(NewCmdCreateWallet(w, f))
	cmd.AddCommand(NewCmdDeleteWallet(w, f))
	cmd.AddCommand(NewCmdDescribeWallet(w, f))
	cmd.AddCommand(NewCmdImportWallet(w, f))
	cmd.AddCommand(NewCmdListWallets(w, f))
	cmd.AddCommand(NewCmdRenameWallet(w, f))

	return cmd
}

type RootFlags struct {
	Output string
	Home   string
}

func (f *RootFlags) Validate() error {
	return flags.ValidateOutput(f.Output)
}
