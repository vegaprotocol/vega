package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jessevdk/go-flags"
)

type NodeWalletCmd struct {
	// Global options
	config.RootPathFlag
	config.PassphraseFlag

	// Subcommands
	Import nodeWalletImport `command:"import"`
	Verify nodeWalletVerify `command:"verify"`
}

var nodeWalletCmd NodeWalletCmd

func NodeWallet(ctx context.Context, parser *flags.Parser) error {
	root := config.NewRootPathFlag()
	nodeWalletCmd = NodeWalletCmd{
		RootPathFlag: root,
		Import: nodeWalletImport{
			Config: nodewallet.NewDefaultConfig(root.RootPath),
		},
		Verify: nodeWalletVerify{
			Config: nodewallet.NewDefaultConfig(root.RootPath),
		},
	}

	var (
		short = "Manages the node wallet"
		long  = `The nodewallet is a wallet owned by the vega node, it contains all
	the information to login to other wallets from external blockchain that
	vega will need to run properly (e.g and ethereum wallet, which allow vega
	to sign transaction  to be verified on the ethereum blockchain) available
	wallet: eth, vega`
	)

	_, err := parser.AddCommand("nodewallet", short, long, &nodeWalletCmd)
	return err
}

type nodeWalletImport struct {
	Config nodewallet.Config

	WalletPassphrase config.Passphrase `short:"w" long:"wallet-passphrase"`

	Chain      string `short:"c" long:"chain" required:"true"`
	WalletPath string `long:"wallet-path" required:"true"`
	Force      bool   `long:"force"`
}

func (opts *nodeWalletImport) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	if ok, err := fsutil.PathExists(nodeWalletCmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	nodePass, err := nodeWalletCmd.Passphrase.Get("node wallet")
	if err != nil {
		return fmt.Errorf("invalid node wallet passphrase: %w", err)
	}

	walletPass, err := opts.WalletPassphrase.Get("blockchain wallet")
	if err != nil {
		return fmt.Errorf("invalid blockchain wallet passphrase: %w", err)
	}

	conf, err := config.Read(nodeWalletCmd.RootPath)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	// instantiate the ETHClient
	var ethclt eth.ETHClient = nil
	if opts.Config.ETH.Address != "" {
		ethclt, err = ethclient.Dial(opts.Config.ETH.Address)
		if err != nil {
			return fmt.Errorf("failed to connect to Ethereum at %s: %w", opts.Config.ETH.Address, err)
		}
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
}

func (opts *nodeWalletVerify) Execute(_ []string) error {
	if ok, err := fsutil.PathExists(nodeWalletCmd.RootPath); !ok {
		return fmt.Errorf("invalid root directory path: %w", err)
	}

	pass, err := nodeWalletCmd.Passphrase.Get("node wallet")
	if err != nil {
		return err
	}

	conf, err := config.Read(nodeWalletCmd.RootPath)
	if err != nil {
		return err
	}
	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	// instantiate the ETHClient
	var ethclt eth.ETHClient = nil
	if opts.Config.ETH.Address != "" {
		ethclt, err = ethclient.Dial(opts.Config.ETH.Address)
		if err != nil {
			return fmt.Errorf("failed to connect to Ethereum at %s: %w", opts.Config.ETH.Address, err)
		}
	}

	err = nodewallet.Verify(conf.NodeWallet, pass, ethclt)
	if err != nil {
		return err
	}

	fmt.Printf("ok\n")
	return nil
}
