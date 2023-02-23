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

package validators_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"testing"

	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/proto"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	types1 "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTwoNodes(top *testTop) {
	number := 2
	tmPubKeys := make([]string, 0, number)

	for i := 0; i < number; i++ {
		tmPubKeys = append(tmPubKeys, fmt.Sprintf("tm-pub-key-%d", i))
	}

	ctx := context.Background()

	for i := 0; i < number; i++ {
		top.AddNewNode(ctx, &commandspb.AnnounceNode{
			Id:              fmt.Sprintf("vega-master-pubkey-%d", i),
			ChainPubKey:     tmPubKeys[0],
			VegaPubKey:      hexEncode(fmt.Sprintf("vega-key-%d", i)),
			EthereumAddress: fmt.Sprintf("eth-address-%d", i),
		}, validators.ValidatorStatusTendermint)
	}
}

func hexEncode(str string) string {
	return hex.EncodeToString([]byte(str))
}

func TestTopologyCheckpoint(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Run("test checkpoint success", testTopologyCheckpointSuccess)
		t.Run("test checkpoint uses relative block height", testTopologyCheckpointUsesRelativeBlockHeight)
	}
}

func TestCheckPointLoading(t *testing.T) {
	newTop := getTestTopWithDefaultValidator(t)
	defer newTop.ctrl.Finish()
	newTop.timeService.EXPECT().GetTimeNow().AnyTimes()

	inFile := "testcp/20220411202622-135-812dab0eb11196b49fd716329feb50c243f645226460df760168215d73acf0dd.cp"
	data, _ := ioutil.ReadFile(inFile)
	cp := &checkpoint.Checkpoint{}
	if err := proto.Unmarshal(data, cp); err != nil {
		t.Fatal(err)
	}
	require.Equal(t, 1, len(newTop.AllNodeIDs()))
	newTop.Load(context.Background(), cp.Validators)
	require.Equal(t, 2, len(newTop.AllNodeIDs()))
}

func testTopologyCheckpointSuccess(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	ctx := context.Background()
	addTwoNodes(top)

	kr1 := &commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    1,
		TargetBlock:       10,
		NewPubKey:         "new-vega-key",
		CurrentPubKeyHash: hashKey("vega-key-0"),
	}
	err := top.AddKeyRotate(ctx, "vega-master-pubkey-0", 5, kr1)
	assert.NoError(t, err)

	kr2 := &commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    1,
		TargetBlock:       11,
		NewPubKey:         "new-vega-key-1",
		CurrentPubKeyHash: hashKey("vega-key-1"),
	}
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-1", 5, kr2)
	assert.NoError(t, err)

	ekr1 := newEthereumKeyRotationSubmission("eth-address-0", "new-eth-address-0", 10, "")
	err = top.ProcessEthereumKeyRotation(ctx, hexEncode("vega-key-0"), ekr1, MockVerify)
	assert.NoError(t, err)

	ekr2 := newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 11, "")
	err = top.ProcessEthereumKeyRotation(ctx, hexEncode("vega-key-1"), ekr2, MockVerify)
	assert.NoError(t, err)

	pkrs := top.GetAllPendingKeyRotations()
	assert.Len(t, pkrs, 2)

	ckp, err := top.Checkpoint()
	assert.NotEmpty(t, ckp)
	assert.NoError(t, err)

	newTop := getTestTopWithDefaultValidator(t)
	defer newTop.ctrl.Finish()

	addTwoNodes(newTop)
	newTop.Load(ctx, ckp)

	newPkrs := newTop.GetAllPendingKeyRotations()
	assert.Len(t, newPkrs, 2)
	assert.Equal(t, pkrs, newPkrs)

	assert.Equal(t, top.GetPendingEthereumKeyRotation(ekr1.TargetBlock, "vega-master-pubkey-0"), newTop.GetPendingEthereumKeyRotation(ekr1.TargetBlock, "vega-master-pubkey-0"))
	assert.Equal(t, top.GetPendingEthereumKeyRotation(ekr2.TargetBlock, "vega-master-pubkey-1"), newTop.GetPendingEthereumKeyRotation(ekr2.TargetBlock, "vega-master-pubkey-1"))
}

func testTopologyCheckpointUsesRelativeBlockHeight(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	ctx := context.Background()
	addTwoNodes(top)

	kr1 := &commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    1,
		TargetBlock:       105,
		NewPubKey:         "new-vega-key",
		CurrentPubKeyHash: hashKey("vega-key-0"),
	}
	err := top.AddKeyRotate(ctx, "vega-master-pubkey-0", 5, kr1)
	assert.NoError(t, err)

	kr2 := &commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    1,
		TargetBlock:       115,
		NewPubKey:         "new-vega-key-1",
		CurrentPubKeyHash: hashKey("vega-key-1"),
	}
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-1", 5, kr2)
	assert.NoError(t, err)

	ekr1 := newEthereumKeyRotationSubmission("eth-address-0", "new-eth-address-0", 105, "")
	err = top.ProcessEthereumKeyRotation(ctx, hexEncode("vega-key-0"), ekr1, MockVerify)
	assert.NoError(t, err)

	ekr2 := newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 115, "")
	err = top.ProcessEthereumKeyRotation(ctx, hexEncode("vega-key-1"), ekr2, MockVerify)
	assert.NoError(t, err)

	pkrs := top.GetAllPendingKeyRotations()
	assert.Len(t, pkrs, 2)

	ckp, err := top.Checkpoint()
	assert.NotEmpty(t, ckp)
	assert.NoError(t, err)

	newTop := getTestTopWithDefaultValidator(t)
	defer newTop.ctrl.Finish()
	newTop.timeService.EXPECT().GetTimeNow().AnyTimes()

	addTwoNodes(newTop)

	var newNetworkBlockHeight uint64 = 100

	// set current block height to newNetworkBlockHeight
	newTop.BeginBlock(ctx, abcitypes.RequestBeginBlock{Header: types1.Header{Height: int64(newNetworkBlockHeight)}})

	newTop.Load(ctx, ckp)

	newPkrs := newTop.GetAllPendingKeyRotations()
	assert.Len(t, newPkrs, 2)

	assert.Equal(t, pkrs[0].BlockHeight+newNetworkBlockHeight, newPkrs[0].BlockHeight)
	assert.Equal(t, pkrs[1].BlockHeight+newNetworkBlockHeight, newPkrs[1].BlockHeight)
}
