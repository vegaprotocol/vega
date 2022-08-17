package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/paths"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	netstore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DefaultForwarderRetryCount = 5
	ForwarderRequestTimeout    = 5 * time.Second
)

type Error struct {
	Err string `json:"error"`
}

type Writer struct {
	Out io.Writer
	Err io.Writer
}

func Execute(w *Writer) {
	c := NewCmdRoot(w.Out)

	execErr := c.Execute()
	if execErr == nil {
		return
	}

	defer os.Exit(1)

	if errors.Is(execErr, flags.ErrUnsupportedOutput) {
		_, _ = fmt.Fprintln(w.Err, execErr)
	}

	output, _ := c.Flags().GetString("output")
	switch output {
	case flags.InteractiveOutput:
		fprintErrorInteractive(w, execErr)
	case flags.JSONOutput:
		fprintErrorJSON(w.Err, execErr)
	}
}

func fprintErrorInteractive(w *Writer, execErr error) {
	if vgterm.HasTTY() {
		p := printer.NewInteractivePrinter(w.Out)
		p.Print(p.String().CrossMark().DangerText(execErr.Error()).NextLine())
	} else {
		_, _ = fmt.Fprintln(w.Err, execErr)
	}
}

func fprintErrorJSON(w io.Writer, err error) {
	jsonErr := printer.FprintJSON(w, Error{
		Err: err.Error(),
	})
	if jsonErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "couldn't format error as JSON: %v\n", jsonErr)
		_, _ = fmt.Fprintf(os.Stderr, "original error: %v\n", err)
	}
}

func autoCompleteWallet(cmd *cobra.Command, vegaHome string) {
	err := cmd.RegisterFlagCompletionFunc("wallet", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		s, err := wallets.InitialiseStore(vegaHome)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		ws, err := wallet.ListWallets(s)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		return ws.Wallets, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		panic(err)
	}
}

func autoCompleteNetwork(cmd *cobra.Command, vegaHome string) {
	err := cmd.RegisterFlagCompletionFunc("network", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		vegaPaths := paths.New(vegaHome)

		netStore, err := netstore.InitialiseStore(vegaPaths)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		nets, err := netStore.ListNetworks()
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		return nets, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		panic(err)
	}
}

func autoCompleteLogLevel(cmd *cobra.Command) {
	err := cmd.RegisterFlagCompletionFunc("level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return SupportedLogLevels, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		panic(err)
	}
}

func getNetworkVersion(url string) (string, error) {
	connection, err := grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("couldn't initialize gRPC client: %w", err)
	}

	client := apipb.NewCoreServiceClient(connection)
	timeout, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	statistics, err := client.Statistics(timeout, &apipb.StatisticsRequest{})
	if err != nil {
		return "", fmt.Errorf("couldn't get network statistics: %w", err)
	}
	return statistics.Statistics.AppVersion, nil
}
