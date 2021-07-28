package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type FaucetCmd struct {
	Init faucetInit `command:"init" description:"Generates the faucet configuration"`
	Run  faucetRun  `command:"run" description:"Runs the faucet"`
	Help bool       `short:"h" long:"help" description:"Show this help message"`
}

// faucetCmd is a global variable that holds generic options for the faucet
// sub-commands.
var faucetCmd FaucetCmd

func Faucet(ctx context.Context, parser *flags.Parser) error {
	defaultPath := config.NewRootPathFlag()
	faucetCmd = FaucetCmd{
		Init: faucetInit{
			RootPathFlag: defaultPath,
		},
		Run: faucetRun{
			ctx:          ctx,
			RootPathFlag: defaultPath,
			Config:       faucet.NewDefaultConfig(defaultPath.RootPath),
		},
	}

	_, err := parser.AddCommand("faucet", "Allow deposit of builtin asset", "", &faucetCmd)
	return err
}

type faucetInit struct {
	config.RootPathFlag
	config.PassphraseFlag
	Force bool `short:"f" long:"force" description:"Erase existing configuration at specified path"`
	Help  bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *faucetInit) Execute(_ []string) error {
	if opts.Help {
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: "vega faucet init subcommand help",
		}
	}

	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	pass, err := opts.PassphraseFile.Get("faucet")
	if err != nil {
		return err
	}

	pubkey, err := faucet.GenConfig(log, opts.RootPath, pass, opts.Force)
	if err != nil {
		return err
	}
	fmt.Printf("pubkey: %s\n", pubkey)
	return nil
}

type faucetRun struct {
	ctx context.Context

	config.RootPathFlag
	config.PassphraseFlag

	faucet.Config

	Help bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *faucetRun) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	cfg, err := faucet.LoadConfig(opts.RootPath)
	if err != nil {
		return err
	}
	opts.Config = *cfg
	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	pass, err := opts.PassphraseFile.Get("faucet")
	if err != nil {
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
