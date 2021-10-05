package main

import (
	"context"
	"fmt"
	"io/ioutil"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/txn"

	"github.com/jessevdk/go-flags"
)

type CheckpointCmd struct {
	// Global variables
	config.VegaHomeFlag
	// wallet config flags
	config.PassphraseFlag
	// Subcommands.
	Restore checkpointRestore `command:"restore"`
}

type checkpointRestore struct {
	nodewallet.Config
	// opts for command
	CPFile string `short:"f" long:"checkpoint-file" description:"name of the file containing the checkpoint data"`
}

var checkpointCmd CheckpointCmd

// Checkpoint - This function is invoked from `Register` in main.go
func Checkpoint(ctx context.Context, parser *flags.Parser) error {

	// here we initialize the global exampleCmd with needed default values.
	checkpointCmd = CheckpointCmd{
		Restore: checkpointRestore{},
	}
	_, err := parser.AddCommand("checkpoint", "Restore checkpoint", "Submits restore transaction to the chain to quickly restart the node from a given state", &checkpointCmd)
	return err
}

func (c *checkpointRestore) Execute(args []string) error {
	if c.CPFile == "" {
		return fmt.Errorf("no file specified")
	}
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	exists, err := vgfs.FileExists(c.CPFile)
	if err != nil {
		return fmt.Errorf("couldn't verify file presence at %s: %w", c.CPFile, err)
	}
	if !exists {
		return fmt.Errorf("checkpoint file not found: %w", err)
	}

	cpData, err := ioutil.ReadFile(c.CPFile)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint file: %w", err)
	}
	commander, err := getNodeWalletCommander(log)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}

	cmd := &commandspb.RestoreSnapshot{
		Data: cpData,
	}
	ch := make(chan error)
	commander.Command(context.Background(), txn.CheckpointRestoreCommand, cmd, func(ok bool) {
		if !ok {
			ch <- fmt.Errorf("failed to send restore command")
		}
		close(ch)
	})
	return <-ch
}

func getNodeWalletCommander(log *logging.Logger) (*nodewallet.Commander, error) {
	vegaPaths := paths.NewPaths(checkpointCmd.VegaHome)

	_, cfg, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return nil, err
	}

	registryPass, err := checkpointCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return nil, err
	}

	vegaWallet, err := nodewallet.GetVegaWallet(vegaPaths, registryPass)
	if err != nil {
		return nil, fmt.Errorf("couldn't get Vega node wallet: %w", err)
	}

	statistics := stats.New(log, cfg.Stats, CLIVersion, CLIVersionHash)
	abciClient, err := abci.NewClient(cfg.Blockchain.Tendermint.ClientAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise ABCI client: %w", err)
	}

	commander, err := nodewallet.NewCommander(log, nil, vegaWallet, statistics)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet commander: %w", err)
	}
	
	commander.SetChain(blockchain.NewClient(abciClient))
	return commander, nil
}
