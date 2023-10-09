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

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/jessevdk/go-flags"
)

type RotateEthKeyCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"passphrase-file"`

	TargetBlock      uint64 `description:"The future block height at which the rotation will take place" long:"target-block"      short:"b"`
	RotateFrom       string `description:"Ethereum address being rotated from"                           long:"rotate-from"       short:"r"`
	SubmitterAddress string `description:"Ethereum address to use as a submitter to contract changes"    long:"submitter-address" short:"s"`
}

var rotateEthKeyCmd RotateEthKeyCmd

func (opts *RotateEthKeyCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer log.AtExit()

	output, err := opts.GetOutput()
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

	registryPass, err := opts.Get("node wallet", false)
	if err != nil {
		return err
	}

	nodeWallets, err := nodewallets.GetNodeWallets(conf.NodeWallet, vegaPaths, registryPass)
	if err != nil {
		return fmt.Errorf("couldn't get node wallets: %w", err)
	}

	// ensure the nodewallet is setup properly, if not we could not complete the command
	if err := nodeWallets.Verify(); err != nil {
		return fmt.Errorf("node wallet misconfigured: %w", err)
	}

	cmd := commandspb.EthereumKeyRotateSubmission{
		CurrentAddress:   crypto.EthereumChecksumAddress(opts.RotateFrom),
		NewAddress:       nodeWallets.Ethereum.PubKey().Hex(),
		TargetBlock:      opts.TargetBlock,
		SubmitterAddress: opts.SubmitterAddress,
	}

	if len(cmd.SubmitterAddress) != 0 {
		cmd.SubmitterAddress = crypto.EthereumChecksumAddress(cmd.SubmitterAddress)
	}

	if err := validators.SignEthereumKeyRotation(&cmd, nodeWallets.Ethereum); err != nil {
		return err
	}

	commander, _, cfunc, err := getNodeWalletCommander(log, registryPass, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}
	defer cfunc()

	var txHash string
	ch := make(chan struct{})
	commander.Command(
		context.Background(),
		txn.RotateEthereumKeySubmissionCommand,
		&cmd,
		func(h string, e error) {
			txHash, err = h, e
			close(ch)
		}, nil)

	<-ch
	if err != nil {
		return err
	}

	if output.IsHuman() {
		fmt.Printf("ethereum key rotation successfully sent\ntxHash: %s\nethereum signature: %v\nRotating from: %s\nRotating to: %s",
			txHash,
			cmd.EthereumSignature.Value,
			opts.RotateFrom,
			cmd.NewAddress,
		)
	} else if output.IsJSON() {
		return vgjson.Print(struct {
			TxHash            string `json:"txHash"`
			EthereumSignature string `json:"ethereumSignature"`
			RotateFrom        string `json:"rotateFrom"`
			RotateTo          string `json:"rotateTo"`
		}{
			TxHash:            txHash,
			RotateFrom:        opts.RotateFrom,
			RotateTo:          cmd.NewAddress,
			EthereumSignature: cmd.EthereumSignature.Value,
		})
	}

	return nil
}

func RotateEthKey(ctx context.Context, parser *flags.Parser) error {
	announceNodeCmd = AnnounceNodeCmd{}

	var (
		short = "Send a transaction to rotate to current ethereum key"
		long  = "Send a transaction to rotate to current ethereum key"
	)
	_, err := parser.AddCommand("rotate_eth_key", short, long, &rotateEthKeyCmd)
	return err
}
