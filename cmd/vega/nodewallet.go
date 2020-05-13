package main

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

type nodeWalletCommand struct {
	command

	rootPath         string
	passphrase       string
	walletPassphrase string
	chain            string
	walletPath       string
	force            bool

	Log *logging.Logger
}

func (w *nodeWalletCommand) Init(c *Cli) {
	w.cli = c
	w.cmd = &cobra.Command{
		Use:   "nodewallet",
		Short: "The nodewallet subcommand",
		Long:  "The nodewallet is a wallet owned by the vega node, it will contains all the informations to login to other wallets from external blockchain that vega will need to run properly (e.g and ethereum wallet, which allow vega to sign transaction  to be verified on the ethereum blockchain) available wallet: eth, vega",
	}

	imprt := &cobra.Command{
		Use:   "import",
		Short: "Import a new wallet",
		Long:  "Import a new wallet",
		RunE:  w.Import,
	}
	imprt.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	imprt.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "The passphrase use to unlock the vega nodewallet")
	imprt.Flags().StringVarP(&w.walletPassphrase, "wallet-passphrase", "w", "", "The passphrase used to unlock the target blockchain wallet to be added to the vega nodewallet (e.g: ethereum blockchain wallet)")
	imprt.Flags().StringVarP(&w.chain, "chain", "c", "", "Name of the blockchain we want to import the wallet for (eth or vega")
	imprt.Flags().StringVarP(&w.walletPath, "wallet-path", "", "", "Path of the wallet to import (needs to be an absolute path)")
	imprt.Flags().BoolVarP(&w.force, "force", "", false, "Force to overwrite an existing wallet import")
	w.cmd.AddCommand(imprt)

	show := &cobra.Command{
		Use:   "show",
		Short: "Show the imported wallets",
		Long:  "Show the imported wallets",
		RunE:  w.Show,
	}
	show.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	show.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "The passphrase used to unlock the vega nodewallet")
	w.cmd.AddCommand(show)

	verify := &cobra.Command{
		Use:   "verify",
		Short: "Verify a nodewallet",
		Long:  "Verify a nodewallet, try to load the nodewallet using the passphrase, then try to load all the wallet save in it.",
		RunE:  w.Verify,
	}
	verify.Flags().StringVarP(&w.rootPath, "root-path", "r", fsutil.DefaultVegaDir(), "Path of the root directory in which the configuration will be located")
	verify.Flags().StringVarP(&w.passphrase, "passphrase", "p", "", "The passphrase used to unlock the vega nodewallet")
	w.cmd.AddCommand(verify)

}

func (w *nodeWalletCommand) Import(cmd *cobra.Command, args []string) error {
	if len(w.walletPassphrase) <= 0 {
		return errors.New("passphrase is required")
	}
	if len(w.passphrase) <= 0 {
		return errors.New("wallet-passphrase is required")
	}
	if len(w.chain) <= 0 {
		return errors.New("chain is required")
	}
	if len(w.walletPath) <= 0 {
		return errors.New("wallet-path is required")
	}

	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	conf, err := config.Read(w.rootPath)
	if err != nil {
		return err
	}

	err = nodewallet.IsSupported(w.chain)
	if err != nil {
		return err
	}

	// instanciate the ETHClient
	ethclt, err := ethclient.Dial(conf.NodeWallet.ETH.Address)
	if err != nil {
		return err
	}

	nw, err := nodewallet.New(w.Log, conf.NodeWallet, w.passphrase, ethclt)
	if err != nil {
		return err
	}

	_, ok := nw.Get(nodewallet.Blockchain(w.chain))
	if ok && w.force {
		w.Log.Warn("a wallet is already imported for the current chain, this action will rewrite the import", logging.String("chain", w.chain))
	} else if ok {
		return fmt.Errorf("a wallet is already imported for the chain %v, please rerun with option --force to overwrite it", w.chain)
	}

	err = nw.Import(w.chain, w.passphrase, w.walletPassphrase, w.walletPath)
	if err != nil {
		return err
	}

	fmt.Println("import success")
	return nil
}

func (w *nodeWalletCommand) Verify(cmd *cobra.Command, args []string) error {
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}
	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}

	conf, err := config.Read(w.rootPath)
	if err != nil {
		return err
	}

	// instanciate the ETHClient
	ethclt, err := ethclient.Dial(conf.NodeWallet.ETH.Address)
	if err != nil {
		return err
	}

	err = nodewallet.Verify(conf.NodeWallet, w.passphrase, ethclt)
	if err != nil {
		return err
	}

	fmt.Printf("ok\n")

	return nil
}

func (w *nodeWalletCommand) Show(cmd *cobra.Command, args []string) error {
	if len(w.passphrase) <= 0 {
		return errors.New("passphrase is required")
	}
	if ok, err := fsutil.PathExists(w.rootPath); !ok {
		return fmt.Errorf("invalid root directory path: %v", err)
	}
	conf, err := config.Read(w.rootPath)
	if err != nil {
		return err
	}

	// instanciate the ETHClient
	ethclt, err := ethclient.Dial(conf.NodeWallet.ETH.Address)
	if err != nil {
		return err
	}

	nw, err := nodewallet.New(w.Log, conf.NodeWallet, w.passphrase, ethclt)
	if err != nil {
		return err
	}
	s, err := nw.Dump()
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", s)

	return nil
}
