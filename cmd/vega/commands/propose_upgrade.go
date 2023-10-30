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
	"strings"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/protocolupgrade"
	"code.vegaprotocol.io/vega/core/txn"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/blang/semver"
	"github.com/jessevdk/go-flags"
)

type ProposeUpgradeCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"passphrase-file"`

	VegaReleaseTag     string `description:"A valid vega core release tag for the upgrade proposal" long:"vega-release-tag" required:"true" short:"v"`
	UpgradeBlockHeight uint64 `description:"The block height at which the upgrade should be made"   long:"height"           required:"true" short:"h"`
}

var proposeUpgradeCmd ProposeUpgradeCmd

func (opts *ProposeUpgradeCmd) Execute(_ []string) error {
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

	if !strings.HasPrefix(opts.VegaReleaseTag, "v") {
		return errors.New("invalid vega release tag, expected prefix 'v' (example: v0.71.9)")
	}

	cmd := commandspb.ProtocolUpgradeProposal{
		VegaReleaseTag:     opts.VegaReleaseTag,
		UpgradeBlockHeight: opts.UpgradeBlockHeight,
	}

	commander, blockData, cfunc, err := getNodeWalletCommander(log, registryPass, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to get commander: %w", err)
	}

	if opts.UpgradeBlockHeight <= blockData.Height {
		return fmt.Errorf("upgrade block earlier than current block height")
	}

	_, err = semver.Parse(protocolupgrade.TrimReleaseTag(opts.VegaReleaseTag))
	if err != nil {
		return fmt.Errorf("invalid protocol version for upgrade received: version (%s), %w", opts.VegaReleaseTag, err)
	}

	defer cfunc()

	tid := vgcrypto.RandomHash()
	powNonce, _, err := vgcrypto.PoW(blockData.Hash, tid, uint(blockData.SpamPowDifficulty), vgcrypto.Sha3)
	if err != nil {
		return fmt.Errorf("failed to get proof of work: %w", err)
	}

	pow := &commandspb.ProofOfWork{
		Tid:   tid,
		Nonce: powNonce,
	}

	var txHash string
	ch := make(chan struct{})
	commander.CommandWithPoW(
		context.Background(),
		txn.ProtocolUpgradeCommand,
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
		fmt.Printf("Upgrade proposal sent.\ntxHash: %s", txHash)
	} else if output.IsJSON() {
		return vgjson.Print(struct {
			TxHash string `json:"txHash"`
		}{
			TxHash: txHash,
		})
	}
	return err
}

func ProposeProtocolUpgrade(ctx context.Context, parser *flags.Parser) error {
	proposeUpgradeCmd = ProposeUpgradeCmd{}

	var (
		short = "Propose a protocol upgrade"
		long  = "Propose a protocol upgrade"
	)
	_, err := parser.AddCommand("protocol_upgrade_proposal", short, long, &proposeUpgradeCmd)
	return err
}
