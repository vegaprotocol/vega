package staking_test

import (
	"bytes"
	"context"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

var allKey = (&types.PayloadStakingAccounts{}).Key()

func TestAccountsSnapshotEmpty(t *testing.T) {
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()

	h, err := acc.GetHash(allKey)
	require.Nil(t, err)
	require.NotNil(t, h)
}

func TestAccountsSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	acc := getAccountingTest(t)
	defer acc.ctrl.Finish()
	acc.broker.EXPECT().Send(gomock.Any()).Times(1)

	h1, err := acc.GetHash(allKey)
	require.Nil(t, err)

	evt := &types.StakeLinking{
		ID:     "someid1",
		Type:   types.StakeLinkingTypeDeposited,
		TS:     100,
		Party:  testParty,
		Amount: num.NewUint(10),
	}
	acc.AddEvent(ctx, evt)

	// Check hash has change now an event as been added
	h2, err := acc.GetHash(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))

	// Get state ready to load in a new instance of the engine
	state, _, err := acc.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapAcc := getAccountingTest(t)
	defer snapAcc.ctrl.Finish()

	// Load it in anc check that the accounts and their balances have returned
	provs, err := snapAcc.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)
	require.Nil(t, provs)
	require.Equal(t, acc.GetAllAvailableBalances(), snapAcc.GetAllAvailableBalances())
}
