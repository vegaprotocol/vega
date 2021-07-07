package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/storage"

	"github.com/jessevdk/go-flags"
	"github.com/zannen/toml"
)

type InitCmd struct {
	config.RootPathFlag

	// We've unified the passphrase flag as config.PassphraseFlag, which uses --passphrase.
	// As systemtests uses --nodewallet-passphrase we'll define the flag directly here
	// TODO: uncomment this line and remove the Passphrase field.
	// config.PassphraseFlag
	Passphrase config.Passphrase `short:"p" long:"nodewallet-passphrase" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`

	Force      bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`
	GenBuiltin bool `short:"b" long:"gen-builtinasset-faucet" description:"Generate the builtin asset configuration (not for production)"`

	Help bool `short:"h" long:"help" description:"Show this help message"`
}

var initCmd InitCmd

func (opts *InitCmd) Execute(_ []string) error {
	if opts.Help {
		return &flags.Error{Type: flags.ErrHelp, Message: "vega init subcommand help"}
	}
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	rootPathExists, err := fsutil.PathExists(opts.RootPath)
	if err != nil {
		if _, ok := err.(*fsutil.PathNotFound); !ok {
			return err
		}
	}

	if rootPathExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%v` please remove it first or re-run using -f", opts.RootPath)
	}

	if rootPathExists && opts.Force {
		logger.Info("removing existing configuration", logging.String("path", opts.RootPath))
		os.RemoveAll(opts.RootPath) // ignore any errors here to force removal
	}

	// create the root
	if err = fsutil.EnsureDir(opts.RootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(opts.RootPath, storage.CandlesDataPath)
	fullOrderStorePath := filepath.Join(opts.RootPath, storage.OrdersDataPath)
	fullTradeStorePath := filepath.Join(opts.RootPath, storage.TradesDataPath)
	fullMarketStorePath := filepath.Join(opts.RootPath, storage.MarketsDataPath)

	// create sub-folders
	if err = fsutil.EnsureDir(fullCandleStorePath); err != nil {
		return err
	}
	if err = fsutil.EnsureDir(fullOrderStorePath); err != nil {
		return err
	}
	if err = fsutil.EnsureDir(fullTradeStorePath); err != nil {
		return err
	}
	if err = fsutil.EnsureDir(fullMarketStorePath); err != nil {
		return err
	}

	// generate a default configuration
	cfg := config.NewDefaultConfig(opts.RootPath)

	pass, err := opts.Passphrase.Get("nodewallet")
	if err != nil {
		return err
	}

	// initialize the faucet if needed
	if opts.GenBuiltin {
		pubkey, err := faucet.GenConfig(logger, opts.RootPath, pass, false)
		if err != nil {
			return err
		}
		// add the pubkey to the allowlist
		cfg.EvtForward.BlockchainQueueAllowlist = append(
			cfg.EvtForward.BlockchainQueueAllowlist, pubkey)
	}

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err = toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(filepath.Join(opts.RootPath, "config.toml"))
	if err != nil {
		return err
	}

	if _, err = f.WriteString(buf.String()); err != nil {
		return err
	}

	if err := nodewallet.Initialise(opts.RootPath, pass); err != nil {
		return err
	}

	logger.Info("configuration generated successfully", logging.String("path", opts.RootPath))

	return nil
}

func nodeWalletInit(cfg config.Config, nodeWalletPassphrase string, genDevNodeWallet bool) error {
	if genDevNodeWallet {
		return nodewallet.DevInit(
			cfg.NodeWallet.StorePath,
			cfg.NodeWallet.DevWalletsPath,
			nodeWalletPassphrase,
		)
	}
	return nodewallet.Init(
		cfg.NodeWallet.StorePath,
		nodeWalletPassphrase,
	)
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{
		RootPathFlag: config.NewRootPathFlag(),
	}

	var (
		short = "Initializes a vega node"
		long  = "Generate the minimal configuration required for a vega node to start"
	)
	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
