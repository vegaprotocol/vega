package main

import (
	"context"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"

	"github.com/spf13/cobra"
)

type faucetCommand struct {
	command

	log        *logging.Logger
	rootPath   string
	passphrase string
}

func (f *faucetCommand) Init(c *Cli) {
	f.cli = c
	f.cmd = &cobra.Command{
		Use:   "faucet",
		Short: "The faucet subcommand",
		Long:  "Allow deposit of builtin asset",
	}

	run := &cobra.Command{
		Use:   "run",
		Short: "Run the faucet",
		RunE:  f.Run,
	}

	run.Flags().StringVarP(&f.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	run.Flags().StringVarP(&f.passphrase, "passphrase", "p", "", "Passphrase to access the faucet wallet")
	f.cmd.AddCommand(run)

}

func (f *faucetCommand) Run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		passphrase string
		err        error
	)
	if len(f.passphrase) <= 0 {
		passphrase, err = getTerminalPassphrase()
	} else {
		passphrase, err = getFilePassphrase(f.passphrase)
	}

	if err != nil {
		return err
	}

	cfgwatchr, err := config.NewFromFile(ctx, f.log, f.rootPath, f.rootPath)
	if err != nil {
		f.log.Error("unable to start config watcher", logging.Error(err))
		return err
	}
	conf := cfgwatchr.Get()

	fct, err := faucet.New(f.log, conf.Faucet, passphrase)
	if err != nil {
		return err
	}
	go func() {
		defer cancel()
		err := fct.Start()
		if err != nil {
			f.log.Error("error starting faucet server", logging.Error(err))
		}
	}()

	waitSig(ctx, f.log)

	err = fct.Stop()
	if err != nil {
		f.log.Error("error stopping faucet server", logging.Error(err))
	} else {
		f.log.Info("faucet server stopped with success")
	}

	return nil

}
