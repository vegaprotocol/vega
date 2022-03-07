package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"time"

	api "code.vegaprotocol.io/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/txn"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
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
	nodewallets.Config
	// opts for command
	CPFile string `short:"f" long:"checkpoint-file" description:"name of the file containing the checkpoint data"`
}

var checkpointCmd CheckpointCmd

// Checkpoint - This function is invoked from `Register` in main.go.
func Checkpoint(ctx context.Context, parser *flags.Parser) error {
	// here we initialize the global exampleCmd with needed default values.
	checkpointCmd = CheckpointCmd{
		Restore: checkpointRestore{},
	}
	_, err := parser.AddCommand("checkpoint", "Restore checkpoint", "Submits restore transaction to the chain to quickly restart the node from a given state", &checkpointCmd)
	return err
}

func (c *checkpointRestore) Execute(_ []string) error {
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
	commander, cfunc, err := getNodeWalletCommander(log)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}
	defer cfunc()

	cmd := &commandspb.RestoreSnapshot{
		Data: cpData,
	}

	ch := make(chan error)
	commander.CommandSync(context.Background(), txn.CheckpointRestoreCommand, cmd, func(err error) {
		if err != nil {
			ch <- fmt.Errorf("failed to send restore command: %v", err)
		}
		close(ch)
	}, nil)
	return <-ch
}

func getNodeWalletCommander(log *logging.Logger) (*nodewallets.Commander, context.CancelFunc, error) {
	vegaPaths := paths.New(checkpointCmd.VegaHome)

	_, cfg, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return nil, nil, err
	}

	registryPass, err := checkpointCmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return nil, nil, err
	}

	vegaWallet, err := nodewallets.GetVegaWallet(vegaPaths, registryPass)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get Vega node wallet: %w", err)
	}

	abciClient, err := abci.NewClient(cfg.Blockchain.Tendermint.ClientAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise ABCI client: %w", err)
	}

	coreClient, err := getCoreClient(
		net.JoinHostPort(cfg.API.IP, strconv.Itoa(cfg.API.Port)))
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't connect to node: %w", err)
	}

	ctx, cancel := timeoutContext()
	resp, err := coreClient.LastBlockHeight(ctx, &api.LastBlockHeightRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get last block height: %w", err)
	}

	commander, err := nodewallets.NewCommander(cfg.NodeWallet, log, nil, vegaWallet, heightProvider(resp.Height))
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise node wallet commander: %w", err)
	}

	commander.SetChain(blockchain.NewClient(abciClient))
	return commander, cancel, nil
}

type heightProvider uint64

func (h heightProvider) Height() uint64 {
	return uint64(h)
}

func getCoreClient(address string) (api.CoreServiceClient, error) {
	tdconn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return api.NewCoreServiceClient(tdconn), nil
}

func timeoutContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
