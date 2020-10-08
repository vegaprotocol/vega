package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"github.com/ethereum/go-ethereum/ethclient"
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
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

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

	// instanciate the ETHClient
	ethclt, err := ethclient.Dial(opts.Config.ETH.Address)
	if err != nil {
		return err
	}

	nw, err := nodewallet.New(log, conf.NodeWallet, nodePass, ethclt)
	if err != nil {
		return err
	}

	_, ok := nw.Get(nodewallet.Blockchain(opts.Chain))
	if ok && opts.Force {
		log.Warn("a wallet is already imported for the current chain, this action will rewrite the import", logging.String("chain", opts.Chain))
	} else if ok {
		return fmt.Errorf("a wallet is already imported for the chain %v, please rerun with option --force to overwrite it", opts.Chain)
	}

	err = nw.Import(opts.Chain, nodePass, walletPass, opts.WalletPath)
	if err != nil {
		return err
	}

	fmt.Println("import success")
	return nil
}

type nodeWalletVerify struct {
	Config nodewallet.Config

	RootPathOption
	PassphraseOption
}

func (opts *nodeWalletVerify) Execute(_ []string) error {
	if ok, err := fsutil.PathExists(opts.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	pass, err := opts.Passphrase.Get("node wallet")
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

	// instanciate the ETHClient
	ethclt, err := ethclient.Dial(conf.NodeWallet.ETH.Address)
	if err != nil {
		return err
	}

	err = nodewallet.Verify(conf.NodeWallet, pass, ethclt)
	if err != nil {
		return err
	}

	fmt.Printf("ok\n")
	return nil
}

type nodeWalletCmd struct {
	Import nodeWalletImport `command:"import"`
	Verify nodeWalletVerify `command:"verify"`
}

func NodeWallet(ctx context.Context, parser *flags.Parser) error {
	root := NewRootPathOption()
	cmd := nodeWalletCmd{
		Import: nodeWalletImport{
			Config:         nodewallet.NewDefaultConfig(root.RootPath),
			RootPathOption: root,
		},
		Verify: nodeWalletVerify{
			Config:         nodewallet.NewDefaultConfig(root.RootPath),
			RootPathOption: root,
		},
	}

	_, err := parser.AddCommand("nodewallet", "", "", &cmd)
	return err
}
