package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jessevdk/go-flags"
)

type faucetInit struct {
	RootPathOption
	PassphraseOption
	Force bool `short:"f" long:"force" description:"Erase existing configuratio at specified path"`
}

func (opts *faucetInit) Execute(_ []string) error {
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	pass, err := opts.Passphrase.Get("faucet")
	if err != nil {
		return err
	}

	pubkey, err := faucet.GenConfig(log, opts.RootPath, pass, opts.Force)
	if err != nil {
		return err
	}
	fmt.Printf("pubkey: %v\n", pubkey)

	return nil
}

type faucetRun struct {
	ctx context.Context
	faucet.Config
	RootPathOption
	PassphraseOption
}

func (opts *faucetRun) Execute(_ []string) error {
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	pass, err := opts.Passphrase.Get("faucet")
	if err != nil {
		return err
	}

	cfg, err := faucet.LoadConfig(opts.RootPath)
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

func Faucet(ctx context.Context, parser *flags.Parser) error {
	cmd, err := parser.AddCommand("faucet", "Allow deposit of builtin asset", "", &Empty{})
	if err != nil {
		return err
	}

	if _, err := cmd.AddCommand("init", "Generates the faucet configuration", "", &faucetInit{
		RootPathOption: NewRootPathOption(),
	}); err != nil {
		return err
	}

	if _, err = cmd.AddCommand("run", "Runs the faucet", "", &faucetRun{
		ctx:            ctx,
		Config:         faucet.NewDefaultConfig(fsutil.DefaultVegaDir()),
		RootPathOption: NewRootPathOption(),
	}); err != nil {
		return err
	}

	return nil
}
