// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nodewallets_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/nodewallets/mocks"
	vgnw "code.vegaprotocol.io/vega/core/nodewallets/vega"
	"code.vegaprotocol.io/vega/core/txn"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/require"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testCommander struct {
	*nodewallets.Commander
	ctx    context.Context
	cfunc  context.CancelFunc
	ctrl   *gomock.Controller
	chain  *mocks.MockChain
	bstats *mocks.MockBlockchainStats
	wallet *vgnw.Wallet
}

func getTestCommander(t *testing.T) *testCommander {
	t.Helper()
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	chain := mocks.NewMockChain(ctrl)
	bstats := mocks.NewMockBlockchainStats(ctrl)
	vegaPaths, _ := vgtesting.NewVegaPaths()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)

	_, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false)
	require.NoError(t, err)
	wallet, err := nodewallets.GetVegaWallet(vegaPaths, registryPass)
	require.NoError(t, err)
	require.NotNil(t, wallet)

	cmd, err := nodewallets.NewCommander(nodewallets.NewDefaultConfig(), logging.NewTestLogger(), chain, wallet, bstats)
	require.NoError(t, err)

	return &testCommander{
		Commander: cmd,
		ctx:       ctx,
		cfunc:     cfunc,
		ctrl:      ctrl,
		chain:     chain,
		bstats:    bstats,
		wallet:    wallet,
	}
}

func TestCommand(t *testing.T) {
	t.Run("Signed command - success", testSignedCommandSuccess)
	t.Run("Signed command - failure", testSignedCommandFailure)
}

func testSignedCommandSuccess(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := txn.NodeVoteCommand
	payload := &commandspb.NodeVote{
		Reference: "test",
	}
	expectedChainID := vgrand.RandomStr(5)

	commander.bstats.EXPECT().Height().AnyTimes().Return(uint64(42))
	commander.chain.EXPECT().GetChainID(gomock.Any()).Times(1).Return(expectedChainID, nil)
	commander.chain.EXPECT().SubmitTransactionSync(gomock.Any(), gomock.Any()).Times(1).Return(&tmctypes.ResultBroadcastTx{}, nil)

	ok := make(chan error)
	commander.Command(context.Background(), cmd, payload, func(_ string, err error) {
		ok <- err
	}, nil)
	assert.NoError(t, <-ok)
}

func testSignedCommandFailure(t *testing.T) {
	commander := getTestCommander(t)
	defer commander.Finish()

	cmd := txn.NodeVoteCommand
	payload := &commandspb.NodeVote{
		Reference: "test",
	}
	commander.bstats.EXPECT().Height().AnyTimes().Return(uint64(42))
	commander.chain.EXPECT().GetChainID(gomock.Any()).Times(1).Return(vgrand.RandomStr(5), nil)
	commander.chain.EXPECT().SubmitTransactionSync(gomock.Any(), gomock.Any()).Times(1).Return(&tmctypes.ResultBroadcastTx{}, assert.AnError)

	ok := make(chan error)
	commander.Command(context.Background(), cmd, payload, func(_ string, err error) {
		ok <- err
	}, nil)
	assert.Error(t, <-ok)
}

func (t *testCommander) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}
