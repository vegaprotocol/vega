package nodewallet

import (
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	vgfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"

	"github.com/jessevdk/go-flags"
)

type importCmd struct {
	config.OutputFlag

	Config nodewallet.Config

	WalletPassphrase config.Passphrase `long:"wallet-passphrase-file"`

	Chain      string `short:"c" long:"chain" required:"true" description:"The chain to be imported (vega, ethereum)"`
	WalletPath string `long:"wallet-path" required:"true" description:"The path to the wallet file to import"`
	Force      bool   `long:"force" description:"Should the command re-write an existing nodewallet file if it exists"`
}

func (opts *importCmd) Execute(_ []string) error {
	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	pass, err := rootCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	walletPass, err := opts.WalletPassphrase.Get("blockchain wallet")
	if err != nil {
		return err
	}

	vegaPaths := paths.NewPaths(rootCmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.NodeWallet

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, nil, vegaPaths)
	if err != nil {
		return err
	}

	_, ok := nw.Get(nodewallet.Blockchain(opts.Chain))
	if ok && opts.Force {
		log.Warn("a wallet is already imported for the current chain, this action will rewrite the import", logging.String("chain", opts.Chain))
	} else if ok {
		return fmt.Errorf("a wallet is already imported for the chain %v, please rerun with option --force to overwrite it", opts.Chain)
	}

	data, err := nw.Import(opts.Chain, pass, walletPass, opts.WalletPath)
	if err != nil {
		return err
	}
	data["configFilePath"] = nw.GetConfigFilePath()

	if output.IsHuman() {
		fmt.Println("import successful:")
		vgfmt.PrettyPrint(data)
	} else if output.IsJSON() {
		if err := vgjson.Print(data); err != nil {
			return err
		}
	}

	return nil
}
