package main

import (
	"context"
	"errors"
	"fmt"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/validators"
	"github.com/jessevdk/go-flags"
)

type AnnounceNodeCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"passphrase-file"`

	InfoURL   string `short:"i" long:"info-url" required:"true" description:"An url to the website / information about this validator"`
	Country   string `short:"c" long:"Country" required:"true" description:"The country from which the validator is operating"`
	Name      string `short:"n" long:"name" required:"true" description:"The name of this validator"`
	AvatarURL string `short:"a" long:"avatar-url" required:"true" description:"A link to an avatar picture for this validator"`
	FromEpoch uint64 `short:"f" long:"from-epoch" required:"true" description:"The epoch from which this validator should be ready to validate blocks" `
}

var announceNodeCmd AnnounceNodeCmd

func (opts *AnnounceNodeCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	registryPass, err := opts.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(opts.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	if !conf.IsValidator() {
		return errors.New("node is not a validator")
	}

	nodeWallets, err := nodewallets.GetNodeWallets(conf.NodeWallet, vegaPaths, registryPass)
	if err != nil {
		return fmt.Errorf("couldn't get node wallets: %w", err)
	}

	// ensure the nodewallet is setup properly, if not we could not complete the command
	if err := nodeWallets.Verify(); err != nil {
		return fmt.Errorf("node wallet misconfigured: %w", err)
	}

	cmd := commandspb.AnnounceNode{
		Id:              nodeWallets.Vega.ID().Hex(),
		VegaPubKey:      nodeWallets.Vega.PubKey().Hex(),
		VegaPubKeyIndex: nodeWallets.Vega.Index(),
		ChainPubKey:     nodeWallets.Tendermint.Pubkey,
		EthereumAddress: nodeWallets.Ethereum.PubKey().Hex(),
		FromEpoch:       opts.FromEpoch,
		InfoUrl:         opts.InfoURL,
		Name:            opts.Name,
		AvatarUrl:       opts.AvatarURL,
		Country:         opts.Country,
	}

	if err := validators.SignAnnounceNode(
		&cmd, nodeWallets.Vega, nodeWallets.Ethereum,
	); err != nil {
		return err
	}

	// now we are OK, send the command

	commander, cfunc, err := getNodeWalletCommander(log, registryPass, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}
	defer cfunc()

	ch := make(chan error)
	commander.CommandSync(
		context.Background(),
		txn.AnnounceNodeCommand,
		&cmd,
		func(err error) {
			if err != nil {
				ch <- fmt.Errorf("failed to send restore command: %v", err)
			}
			close(ch)
		}, nil)

	err = <-ch
	if err != nil {
		return err
	}

	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	if output.IsHuman() {
		fmt.Printf("node successfully registered.\nvega signature: %v\nethereum signature: %v\n", cmd.VegaSignature.Value, cmd.EthereumSignature.Value)
	} else if output.IsJSON() {
		return vgjson.Print(struct {
			EthereumSignature string `json:"ethereumSignature"`
			VegaSignature     string `json:"vegaSignature"`
		}{
			EthereumSignature: cmd.EthereumSignature.Value,
			VegaSignature:     cmd.VegaSignature.Value,
		})
	}

	return nil
}

func AnnounceNode(ctx context.Context, parser *flags.Parser) error {
	announceNodeCmd = AnnounceNodeCmd{}

	var (
		short = "Announce a node as a potential validator to the network"
		long  = "Announce a node as a potential validator to the network"
	)
	_, err := parser.AddCommand("announce_node", short, long, &announceNodeCmd)
	return err
}
