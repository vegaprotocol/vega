package nodewallet

import (
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"

	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/api/socket"
	"code.vegaprotocol.io/vega/config"
	vgfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type reloadCmd struct {
	config.OutputFlag

	Config api.Config

	WalletPassphrase config.Passphrase `long:"wallet-passphrase-file"`

	Chain string `short:"c" long:"chain" required:"true" description:"The chain to be imported" choice:"vega" choice:"ethereum"`
}

func (opts *reloadCmd) Execute(_ []string) error {
	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	vegaPaths := paths.New(rootCmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.API

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	sc := socket.NewSocketClient(log, opts.Config)

	var data map[string]string
	switch opts.Chain {
	case vegaChain, ethereumChain:
		resp, err := sc.NodeWalletReload(opts.Chain)
		if err != nil {
			return fmt.Errorf("failed to reload node wallet: %w", err)
		}

		data = map[string]string{
			"OldWallet": resp.OldWallet.String(),
			"NewWallet": resp.NewWallet.String(),
		}
	default:
		return fmt.Errorf("chain %q is not supported", opts.Chain)
	}

	if output.IsHuman() {
		fmt.Println(green("reload successful:"))
		vgfmt.PrettyPrint(data)
	} else if output.IsJSON() {
		if err := vgjson.Print(data); err != nil {
			return err
		}
	}

	return nil
}
