package delegation

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
)

func TestCheckpointBackwardCompatibility(t *testing.T) {
	t.Run("checkpoint with delegation roundtrip", testCheckpointBridgeWithDelegation)
	t.Run("checkpoint with undelegation roundtrip", testCheckpointBridgeWithUndelegation)
	t.Run("checkpoint with new delegation roundtrip", testCheckpointBridgeWithNewDelegation)
	t.Run("full checkpoint roundtrip", testCheckpointBridgeMultiPartyMultiNode)
}

func testCheckpointBridgeWithDelegation(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(2)

	active := []*types.DelegationEntry{&types.DelegationEntry{
		Party:      "party1",
		Node:       "node1",
		Amount:     num.NewUint(50),
		Undelegate: false,
		EpochSeq:   100,
	}}

	pending := []*types.DelegationEntry{&types.DelegationEntry{
		Party:      "party1",
		Node:       "node1",
		Amount:     num.NewUint(100),
		Undelegate: false,
		EpochSeq:   101,
	}}

	data := &types.DelegateCP{
		Active:  active,
		Pending: pending,
		Auto:    []string{},
	}
	cp, _ := proto.Marshal(data.IntoProto())

	testEngine.engine.Load(context.Background(), cp)
	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 100})
	require.Equal(t, num.NewUint(50), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(50), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(150), testEngine.engine.nextPartyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(150), testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount["node1"])

	// take a new checkpoint from the current state and make sure it matches the old one
	cp2, _ := testEngine.engine.Checkpoint()
	require.True(t, bytes.Equal(cp, cp2))
}

func testCheckpointBridgeWithUndelegation(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(2)

	active := []*types.DelegationEntry{&types.DelegationEntry{
		Party:      "party1",
		Node:       "node1",
		Amount:     num.NewUint(150),
		Undelegate: false,
		EpochSeq:   100,
	}}

	pending := []*types.DelegationEntry{&types.DelegationEntry{
		Party:      "party1",
		Node:       "node1",
		Amount:     num.NewUint(50),
		Undelegate: true,
		EpochSeq:   101,
	}}

	data := &types.DelegateCP{
		Active:  active,
		Pending: pending,
		Auto:    []string{},
	}
	cp, _ := proto.Marshal(data.IntoProto())
	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 100})
	testEngine.engine.Load(context.Background(), cp)

	require.Equal(t, num.NewUint(150), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(150), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(100), testEngine.engine.nextPartyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(100), testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount["node1"])

	// take a new checkpoint from the current state and make sure it matches the old one
	cp2, _ := testEngine.engine.Checkpoint()
	require.True(t, bytes.Equal(cp, cp2))
}

func testCheckpointBridgeWithNewDelegation(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(2)

	active := []*types.DelegationEntry{&types.DelegationEntry{
		Party:      "party1",
		Node:       "node1",
		Amount:     num.NewUint(50),
		Undelegate: false,
		EpochSeq:   100,
	}}

	pending := []*types.DelegationEntry{&types.DelegationEntry{
		Party:      "party1",
		Node:       "node1",
		Amount:     num.NewUint(100),
		Undelegate: false,
		EpochSeq:   101,
	},
		&types.DelegationEntry{
			Party:      "party1",
			Node:       "node2",
			Amount:     num.NewUint(120),
			Undelegate: false,
			EpochSeq:   101,
		}}

	data := &types.DelegateCP{
		Active:  active,
		Pending: pending,
		Auto:    []string{},
	}
	cp, _ := proto.Marshal(data.IntoProto())

	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 100})
	testEngine.engine.Load(context.Background(), cp)

	// take a new checkpoint from the current state and make sure it matches the old one
	cp2, _ := testEngine.engine.Checkpoint()
	require.True(t, bytes.Equal(cp, cp2))

	require.Equal(t, num.NewUint(50), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(50), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, 1, len(testEngine.engine.partyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(270), testEngine.engine.nextPartyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(150), testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(120), testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount["node2"])
}

func testCheckpointBridgeMultiPartyMultiNode(t *testing.T) {
	testEngine := getEngine(t)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(2)

	active := []*types.DelegationEntry{
		&types.DelegationEntry{
			Party:      "party1",
			Node:       "node1",
			Amount:     num.NewUint(10),
			Undelegate: false,
			EpochSeq:   100,
		},
		&types.DelegationEntry{
			Party:      "party1",
			Node:       "node2",
			Amount:     num.NewUint(20),
			Undelegate: false,
			EpochSeq:   100,
		},
		&types.DelegationEntry{
			Party:      "party2",
			Node:       "node1",
			Amount:     num.NewUint(30),
			Undelegate: false,
			EpochSeq:   100,
		},
		&types.DelegationEntry{
			Party:      "party2",
			Node:       "node3",
			Amount:     num.NewUint(40),
			Undelegate: false,
			EpochSeq:   100,
		},
		&types.DelegationEntry{
			Party:      "party3",
			Node:       "node3",
			Amount:     num.NewUint(50),
			Undelegate: false,
			EpochSeq:   100,
		},
		&types.DelegationEntry{
			Party:      "party4",
			Node:       "node4",
			Amount:     num.NewUint(60),
			Undelegate: false,
			EpochSeq:   100,
		}}

	// party1 undelegates all from node1 and moves it to node 3
	// party2 delegates to node2
	pending := []*types.DelegationEntry{
		&types.DelegationEntry{
			Party:      "party1",
			Node:       "node1",
			Amount:     num.NewUint(10),
			Undelegate: true,
			EpochSeq:   101,
		},
		&types.DelegationEntry{
			Party:      "party1",
			Node:       "node3",
			Amount:     num.NewUint(10),
			Undelegate: false,
			EpochSeq:   101,
		},
		&types.DelegationEntry{
			Party:      "party2",
			Node:       "node2",
			Amount:     num.NewUint(50),
			Undelegate: false,
			EpochSeq:   101,
		},
		&types.DelegationEntry{
			Party:      "party3",
			Node:       "node3",
			Amount:     num.NewUint(50),
			Undelegate: true,
			EpochSeq:   101,
		},
		&types.DelegationEntry{
			Party:      "party5",
			Node:       "node5",
			Amount:     num.NewUint(70),
			Undelegate: false,
			EpochSeq:   101,
		},
	}

	data := &types.DelegateCP{
		Active:  active,
		Pending: pending,
		Auto:    []string{},
	}
	cp, _ := proto.Marshal(data.IntoProto())

	testEngine.engine.onEpochEvent(context.Background(), types.Epoch{Seq: 100})
	testEngine.engine.Load(context.Background(), cp)

	// take a new checkpoint from the current state and make sure it matches the old one
	cp2, _ := testEngine.engine.Checkpoint()
	require.True(t, bytes.Equal(cp, cp2))

	require.Equal(t, 4, len(testEngine.engine.partyDelegationState))
	require.Equal(t, num.NewUint(30), testEngine.engine.partyDelegationState["party1"].totalDelegated)
	require.Equal(t, num.NewUint(70), testEngine.engine.partyDelegationState["party2"].totalDelegated)
	require.Equal(t, num.NewUint(50), testEngine.engine.partyDelegationState["party3"].totalDelegated)
	require.Equal(t, num.NewUint(60), testEngine.engine.partyDelegationState["party4"].totalDelegated)

	require.Equal(t, num.NewUint(10), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(20), testEngine.engine.partyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(30), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(40), testEngine.engine.partyDelegationState["party2"].nodeToAmount["node3"])
	require.Equal(t, num.NewUint(50), testEngine.engine.partyDelegationState["party3"].nodeToAmount["node3"])
	require.Equal(t, num.NewUint(60), testEngine.engine.partyDelegationState["party4"].nodeToAmount["node4"])

	// expect party3 to have been gone and party 5 to have been added
	require.Equal(t, 4, len(testEngine.engine.nextPartyDelegationState))
	// party1 moved nomination from node1 to node3 no change in total
	require.Equal(t, num.NewUint(30), testEngine.engine.nextPartyDelegationState["party1"].totalDelegated)
	// party2 added nomination for node 2
	require.Equal(t, num.NewUint(120), testEngine.engine.nextPartyDelegationState["party2"].totalDelegated)
	// party 4 did nothing
	require.Equal(t, num.NewUint(60), testEngine.engine.nextPartyDelegationState["party4"].totalDelegated)
	// party 5 joined with nomination to node5
	require.Equal(t, num.NewUint(70), testEngine.engine.nextPartyDelegationState["party5"].totalDelegated)

	require.Equal(t, 2, len(testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount))
	require.Equal(t, num.NewUint(10), testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount["node3"])
	require.Equal(t, num.NewUint(20), testEngine.engine.nextPartyDelegationState["party1"].nodeToAmount["node2"])
	require.Equal(t, 3, len(testEngine.engine.nextPartyDelegationState["party2"].nodeToAmount))
	require.Equal(t, num.NewUint(30), testEngine.engine.nextPartyDelegationState["party2"].nodeToAmount["node1"])
	require.Equal(t, num.NewUint(50), testEngine.engine.nextPartyDelegationState["party2"].nodeToAmount["node2"])
	require.Equal(t, num.NewUint(40), testEngine.engine.nextPartyDelegationState["party2"].nodeToAmount["node3"])
	require.Equal(t, 1, len(testEngine.engine.nextPartyDelegationState["party4"].nodeToAmount))
	require.Equal(t, num.NewUint(60), testEngine.engine.nextPartyDelegationState["party4"].nodeToAmount["node4"])
	require.Equal(t, 1, len(testEngine.engine.nextPartyDelegationState["party5"].nodeToAmount))
	require.Equal(t, num.NewUint(70), testEngine.engine.nextPartyDelegationState["party5"].nodeToAmount["node5"])
}
