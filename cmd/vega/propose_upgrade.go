// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package main

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/txn"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
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

	VegaReleaseTag     string `short:"v" long:"vega-release-tag" required:"true" description:"A valid vega core release tag for the upgrade proposal"`
	UpgradeBlockHeight uint64 `short:"h" long:"height" required:"true" description:"The block height at which the upgrade should be made"`
}

var proposeUpgradeCmd ProposeUpgradeCmd

func (opts *ProposeUpgradeCmd) Execute(args []string) error {
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
	_, err = semver.Parse(opts.VegaReleaseTag)
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

	ch := make(chan error)
	commander.CommandWithPoW(
		context.Background(),
		txn.AnnounceNodeCommand,
		&cmd,
		func(err error) {
			if err != nil {
				ch <- fmt.Errorf("failed to send protocol upgrade proposal command: %v", err)
			}
			close(ch)
		}, nil, pow)

	err = <-ch
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
