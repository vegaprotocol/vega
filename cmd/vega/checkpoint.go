package main

import (
	"context"
	"fmt"
	"io/ioutil"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/config"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/txn"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jessevdk/go-flags"
)

type CheckpointCmd struct {
	// Global variables
	config.RootPathFlag
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
		RootPathFlag: config.NewRootPathFlag(),
		Restore:      checkpointRestore{},
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

	if ok, err := vgfs.FileExists(c.CPFile); !ok {
		return fmt.Errorf("checkpoint file not found: %w", err)
	}

	cpData, err := ioutil.ReadFile(c.CPFile)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint file: %w", err)
	}
	commander, err := c.getCommander(log)
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

func (c checkpointRestore) getCommander(log *logging.Logger) (*nodewallet.Commander, error) {
	nwConf := nodewallet.NewDefaultConfig()
	// instantiate the ETHClient
	ethclt, err := ethclient.Dial(nwConf.ETH.Address)
	if err != nil {
		return nil, err
	}
	nodePass, err := checkpointCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return nil, err
	}

	// nodewallet
	nodeWallet, err := nodewallet.New(log, nwConf, nodePass, ethclt, checkpointCmd.RootPath)
	if err != nil {
		return nil, err
	}

	// ensure all require wallet are available
	if err := nodeWallet.Verify(); err != nil {
		return nil, err
	}
	stats := stats.New(log, stats.NewDefaultConfig(), CLIVersion, CLIVersionHash)
	wal, _ := nodeWallet.Get(nodewallet.Vega)
	commander, err := nodewallet.NewCommander(log, nil, wal, stats)
	if err != nil {
		return nil, err
	}
	return commander, nil
}
