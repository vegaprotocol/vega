package main

import (
	"context"
	"fmt"

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
	force      bool
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

	init := &cobra.Command{
		Use:   "init",
		Short: "Generate the faucet configuration",
		RunE:  f.CmdInit,
	}

	init.Flags().StringVarP(&f.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	init.Flags().StringVarP(&f.passphrase, "passphrase", "p", "", "Passphrase to access the faucet wallet")
	init.Flags().BoolVarP(&f.force, "force", "f", false, "Erase exiting faucet configuration at the specified path")
	f.cmd.AddCommand(init)

}

func (f *faucetCommand) Run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		passphrase string
		err        error
	)
	if len(f.passphrase) <= 0 {
		passphrase, err = getTerminalPassphrase("faucet")
	} else {
		passphrase, err = getFilePassphrase(f.passphrase)
	}

	if err != nil {
		return err
	}

	cfg, err := faucet.LoadConfig(f.rootPath)
	if err != nil {
		return err
	}
	fct, err := faucet.New(f.log, *cfg, passphrase)
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

func (f *faucetCommand) CmdInit(cmd *cobra.Command, args []string) error {
	if ok, err := fsutil.PathExists(f.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	pubkey, err := faucet.GenConfig(f.log, f.rootPath, f.passphrase, f.force)
	if err != nil {
		return err
	}
	fmt.Printf("pubkey: %v\n", pubkey)
	return nil
}
