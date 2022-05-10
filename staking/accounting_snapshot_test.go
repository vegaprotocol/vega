package staking_test

import (
	"bytes"
	"context"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var allKey = (&types.PayloadStakingAccounts{}).Key()

func TestAccountsSnapshotEmpty(t *testing.T) {
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()

	s, _, err := acc.GetState(allKey)
	require.Nil(t, err)
	require.NotNil(t, s)
}

func TestAccountsSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()
	acc.broker.EXPECT().Send(gomock.Any()).Times(1)

	s1, _, err := acc.GetState(allKey)
	require.Nil(t, err)

	evt := &types.StakeLinking{
		ID:              "someid1",
		Type:            types.StakeLinkingTypeDeposited,
		TS:              100,
		Party:           testParty,
		Amount:          num.NewUint(10),
		BlockHeight:     12,
		BlockTime:       1000002000,
		LogIndex:        100022,
		EthereumAddress: "blah",
	}
	acc.AddEvent(ctx, evt)

	// Check state has change now an event as been added
	s2, _, err := acc.GetState(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	// Get state ready to load in a new instance of the engine
	state, _, err := acc.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapAcc := getAccountingTest(t)
	defer snapAcc.ctrl.Finish()

	// Load it in anc check that the accounts and their balances have returned
	snapAcc.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	provs, err := snapAcc.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)
	require.Nil(t, provs)
	require.Equal(t, acc.GetAllAvailableBalances(), snapAcc.GetAllAvailableBalances())

	s3, _, err := snapAcc.GetState(allKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s2, s3))
}
