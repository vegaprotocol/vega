// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package commands

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/validators"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AnnounceNodeCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"passphrase-file"`

	InfoURL          string `description:"An url to the website / information about this validator"               long:"info-url"          required:"true" short:"i"`
	Country          string `description:"The country from which the validator is operating"                      long:"country"           required:"true" short:"c"`
	Name             string `description:"The name of this validator"                                             long:"name"              required:"true" short:"n"`
	AvatarURL        string `description:"A link to an avatar picture for this validator"                         long:"avatar-url"        required:"true" short:"a"`
	FromEpoch        uint64 `description:"The epoch from which this validator should be ready to validate blocks" long:"from-epoch"        required:"true" short:"f"`
	SubmitterAddress string `description:"Ethereum address to use as a submitter to contract changes"             long:"submitter-address" short:"s"`
}

var announceNodeCmd AnnounceNodeCmd

func (opts *AnnounceNodeCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

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
		Id:               nodeWallets.Vega.ID().Hex(),
		VegaPubKey:       nodeWallets.Vega.PubKey().Hex(),
		VegaPubKeyIndex:  nodeWallets.Vega.Index(),
		ChainPubKey:      nodeWallets.Tendermint.Pubkey,
		EthereumAddress:  vgcrypto.EthereumChecksumAddress(nodeWallets.Ethereum.PubKey().Hex()),
		FromEpoch:        opts.FromEpoch,
		InfoUrl:          opts.InfoURL,
		Name:             opts.Name,
		AvatarUrl:        opts.AvatarURL,
		Country:          opts.Country,
		SubmitterAddress: opts.SubmitterAddress,
	}

	if len(cmd.SubmitterAddress) != 0 {
		cmd.SubmitterAddress = vgcrypto.EthereumChecksumAddress(cmd.SubmitterAddress)
	}

	if err := validators.SignAnnounceNode(
		&cmd, nodeWallets.Vega, nodeWallets.Ethereum,
	); err != nil {
		return err
	}

	// now we are OK, send the command

	commander, blockData, cfunc, err := getNodeWalletCommander(log, registryPass, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}
	defer cfunc()

	tid := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(blockData.Hash, tid, uint(blockData.SpamPowDifficulty), vgcrypto.Sha3)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}

	pow := &commandspb.ProofOfWork{
		Tid:   tid,
		Nonce: powNonce,
	}

	var txHash string
	ch := make(chan struct{})
	commander.CommandWithPoW(
		context.Background(),
		txn.AnnounceNodeCommand,
		&cmd,
		func(h string, e error) {
			txHash, err = h, e
			close(ch)
		}, nil, pow)

	<-ch
	if err != nil {
		return err
	}

	if output.IsHuman() {
		fmt.Printf("node successfully announced.\ntxHash: %s\nvega signature: %v\nethereum signature: %v\n",
			txHash,
			cmd.VegaSignature.Value,
			cmd.EthereumSignature.Value,
		)
	} else if output.IsJSON() {
		return vgjson.Print(struct {
			TxHash            string `json:"txHash"`
			EthereumSignature string `json:"ethereumSignature"`
			VegaSignature     string `json:"vegaSignature"`
		}{
			TxHash:            txHash,
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

func getNodeWalletCommander(log *logging.Logger, registryPass string, vegaPaths paths.Paths) (*nodewallets.Commander, *api.LastBlockHeightResponse, context.CancelFunc, error) {
	_, cfg, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return nil, nil, nil, err
	}

	vegaWallet, err := nodewallets.GetVegaWallet(vegaPaths, registryPass)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't get Vega node wallet: %w", err)
	}

	rpcAddr := cfg.Blockchain.Tendermint.RPCAddr
	if len(rpcAddr) <= 0 {
		rpcAddr = "tcp://127.0.0.1:26657"
		log.Warn("Blockchain.Tendermint.RPCAddr is empty, using default", logging.String("address", rpcAddr))
	}

	abciClient, err := abci.NewClient(rpcAddr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't initialise ABCI client: %w", err)
	}

	coreClient, err := getCoreClient(net.JoinHostPort(cfg.API.IP, strconv.Itoa(cfg.API.Port)))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't connect to node: %w", err)
	}

	ctx, cancel := timeoutContext()
	resp, err := coreClient.LastBlockHeight(ctx, &api.LastBlockHeightRequest{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't get last block height: %w", err)
	}

	commander, err := nodewallets.NewCommander(cfg.NodeWallet, log, blockchain.NewClientWithImpl(abciClient), vegaWallet, heightProvider{height: resp.Height})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't initialise node wallet commander: %w", err)
	}

	return commander, resp, cancel, nil
}

type heightProvider struct {
	height uint64
}

func (h heightProvider) Height() uint64 {
	return h.height
}

func getCoreClient(address string) (api.CoreServiceClient, error) {
	tdconn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return api.NewCoreServiceClient(tdconn), nil
}

func timeoutContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
