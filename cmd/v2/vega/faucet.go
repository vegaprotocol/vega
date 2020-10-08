package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type FaucetCmd struct {
	// Global options
	RootPathOption
	PassphraseOption

	// Subcommands
	Init faucetInit `command:"init" description:"Generates the faucet configuration"`
	Run  faucetRun  `command:"run" description:"Runs the faucet"`
}

// faucetCmd is a global variable that holds generic options for the faucet
// sub-commands.
var faucetCmd FaucetCmd

func Faucet(ctx context.Context, parser *flags.Parser) error {
	defaultPath := NewRootPathOption()
	faucetCmd = FaucetCmd{
		RootPathOption: defaultPath,

		Run: faucetRun{
			ctx:    ctx,
			Config: faucet.NewDefaultConfig(defaultPath.RootPath),
		},
	}

	_, err := parser.AddCommand("faucet", "Allow deposit of builtin asset", "", &faucetCmd)
	return err
}

type faucetInit struct {
	Force bool `short:"f" long:"force" description:"Erase existing configuratio at specified path"`
}

func (opts *faucetInit) Execute(_ []string) error {
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	pass, err := faucetCmd.Passphrase.Get("faucet")
	if err != nil {
		return err
	}

	pubkey, err := faucet.GenConfig(log, faucetCmd.RootPath, pass, opts.Force)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", pubkey)
	return nil
}

type faucetRun struct {
	ctx context.Context
	faucet.Config
}

func (opts *faucetRun) Execute(_ []string) error {
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	pass, err := faucetCmd.Passphrase.Get("faucet")
	if err != nil {
		return err
	}

	cfg, err := faucet.LoadConfig(faucetCmd.RootPath)
	if err != nil {
		return err
	}
	opts.Config = *cfg
	if _, err := flags.Parse(opts); err != nil {
		return err
	}

	f, err := faucet.New(log, opts.Config, pass)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	go func() {
		defer cancel()
		if err := f.Start(); err != nil {
			log.Error("error starting faucet server", logging.Error(err))
		}
	}()

	waitSig(ctx, log)

	if err := f.Stop(); err != nil {
		log.Error("error stopping faucet server", logging.Error(err))
	} else {
		log.Info("faucet server stopped with success")
	}

	return nil
}
