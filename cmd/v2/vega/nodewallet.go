package main

import (
	"context"
	"fmt"
	"log"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/nodewallet"
	"github.com/jessevdk/go-flags"
)

type nodeWalletImport struct {
	Config nodewallet.Config

	RootPathOption
	PassphraseOption
	WalletPassphrase Passphrase `short:"w" long:"wallet-passphrase"`

	Chain      string `short:"c" long:"chain" required:"true"`
	WalletPath string `long:"wallet-path" required:"true"`
	Force      bool   `long:"force"`
}

func (opts *nodeWalletImport) Execute(_ []string) error {
	if ok, err := fsutil.PathExists(opts.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	nodePass, err := opts.Passphrase.Get("node wallet")
	if err != nil {
		return err
	}

	walletPass, err := opts.WalletPassphrase.Get("blockchain wallet")
	if err != nil {
		return err
	}

	conf, err := config.Read(opts.RootPath)
	if err != nil {
		return err
	}
	opts.Config = conf.NodeWallet

	if _, err := flags.Parse(opts); err != nil {
		return err
	}

	log.Printf("opts = %+v\n", opts)
	log.Printf("nodePass = %+v\n", nodePass)
	log.Printf("walletPass = %+v\n", walletPass)
	return nil
}

type nodeCmd struct {
	Import nodeWalletImport `command:"import"`
}

func NodeWallet(ctx context.Context, parser *flags.Parser) error {
	root := NewRootPathOption()
	cmd := nodeCmd{
		Import: nodeWalletImport{
			Config:         nodewallet.NewDefaultConfig(root.RootPath),
			RootPathOption: root,
		},
	}

	_, err := parser.AddCommand("nodewallet", "", "", &cmd)
	return err
}
